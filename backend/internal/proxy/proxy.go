// Package proxy provides functionality to mirror providers from upstream registries.
package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// UpstreamRegistry is the default Terraform registry URL.
const UpstreamRegistry = "https://registry.terraform.io"

// pathComponentRegex validates path components to prevent path traversal attacks.
// Only allows alphanumeric characters, hyphens, underscores, and dots.
// Must start with alphanumeric character.
var pathComponentRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// SanitizePathComponent validates and returns a safe path component.
// Returns an error if the component is invalid or contains path traversal attempts.
func SanitizePathComponent(component string) (string, error) {
	return sanitizePathComponent(component)
}

// sanitizePathComponent validates and returns a safe path component.
// Returns an error if the component is invalid or contains path traversal attempts.
func sanitizePathComponent(component string) (string, error) {
	if component == "" {
		return "", fmt.Errorf("path component cannot be empty")
	}
	// Reject any path traversal attempts
	if strings.Contains(component, "..") ||
		strings.Contains(component, "/") ||
		strings.Contains(component, "\\") ||
		strings.Contains(component, "\x00") {
		return "", fmt.Errorf("path component contains invalid characters")
	}
	// Validate against whitelist regex
	if !pathComponentRegex.MatchString(component) {
		return "", fmt.Errorf("path component contains invalid characters")
	}
	// Return the validated component - at this point it's safe
	return component, nil
}

// BuildSafeProviderPath validates all components and builds a safe directory path.
// Returns the validated path within the storage directory, or an error if any component is invalid.
func BuildSafeProviderPath(storagePath, namespace, name, version, osType, arch string) (string, error) {
	return buildSafeProviderPath(storagePath, namespace, name, version, osType, arch)
}

// buildSafeProviderPath validates all components and builds a safe directory path.
// Returns the validated path within the storage directory, or an error if any component is invalid.
func buildSafeProviderPath(storagePath, namespace, name, version, osType, arch string) (string, error) {
	// Validate each component
	safeNS, err := sanitizePathComponent(namespace)
	if err != nil {
		return "", fmt.Errorf("invalid namespace: %w", err)
	}
	safeName, err := sanitizePathComponent(name)
	if err != nil {
		return "", fmt.Errorf("invalid name: %w", err)
	}
	safeVersion, err := sanitizePathComponent(version)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}
	safeOS, err := sanitizePathComponent(osType)
	if err != nil {
		return "", fmt.Errorf("invalid os: %w", err)
	}
	safeArch, err := sanitizePathComponent(arch)
	if err != nil {
		return "", fmt.Errorf("invalid arch: %w", err)
	}

	// Build path from validated components
	dirPath := filepath.Join(storagePath, safeNS, safeName, safeVersion, safeOS, safeArch)

	// Final safety check: ensure the path is within storage directory
	cleanPath := filepath.Clean(dirPath)
	cleanStorage := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStorage+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes storage directory")
	}

	return cleanPath, nil
}

// SanitizeFilename validates and returns a safe filename.
// Returns an error if the filename is invalid.
func SanitizeFilename(filename string) (string, error) {
	return sanitizeFilename(filename)
}

// sanitizeFilename validates and returns a safe filename.
// Returns an error if the filename is invalid.
func sanitizeFilename(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}
	// Use filepath.Base to extract just the filename
	safeName := filepath.Base(filename)
	if safeName == "." || safeName == ".." || safeName == "" {
		return "", fmt.Errorf("invalid filename")
	}
	// Reject null bytes
	if strings.Contains(safeName, "\x00") {
		return "", fmt.Errorf("filename contains invalid characters")
	}
	return safeName, nil
}

// ProxyService handles provider mirroring operations.
type ProxyService struct {
	httpClient   *http.Client
	storagePath  string
	upstreamURL  string
	proxyURL     string
	proxyType    string
	proxyEnabled bool
	mu           sync.RWMutex
}

// NewProxyService creates a new ProxyService instance.
func NewProxyService(storagePath, upstreamURL string) *ProxyService {
	if upstreamURL == "" {
		upstreamURL = UpstreamRegistry
	}
	return &ProxyService{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		storagePath: storagePath,
		upstreamURL: upstreamURL,
	}
}

// NewProxyServiceWithProxy creates a new ProxyService with proxy support.
func NewProxyServiceWithProxy(storagePath, upstreamURL, proxyURL, proxyType string) *ProxyService {
	if upstreamURL == "" {
		upstreamURL = UpstreamRegistry
	}

	ps := &ProxyService{
		storagePath:  storagePath,
		upstreamURL:  upstreamURL,
		proxyURL:     proxyURL,
		proxyType:    proxyType,
		proxyEnabled: proxyURL != "",
	}
	ps.updateHTTPClient()
	return ps
}

// SetProxy updates the proxy configuration.
func (p *ProxyService) SetProxy(enabled bool, proxyURL, proxyType string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.proxyEnabled = enabled
	p.proxyURL = proxyURL
	p.proxyType = proxyType
	p.updateHTTPClient()
}

// updateHTTPClient creates a new HTTP client with the current proxy settings.
func (p *ProxyService) updateHTTPClient() {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if p.proxyEnabled && p.proxyURL != "" {
		proxyURL := p.proxyURL
		proxyType := strings.ToLower(p.proxyType)

		if proxyType == "socks5" {
			// SOCKS5 proxy
			dialer, err := proxy.SOCKS5("tcp", strings.TrimPrefix(strings.TrimPrefix(proxyURL, "socks5://"), "socks5h://"), nil, proxy.Direct)
			if err == nil {
				transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				}
			}
		} else {
			// HTTP/HTTPS proxy
			if !strings.HasPrefix(proxyURL, "http://") && !strings.HasPrefix(proxyURL, "https://") {
				proxyURL = "http://" + proxyURL
			}
			if proxyU, err := url.Parse(proxyURL); err == nil {
				transport.Proxy = http.ProxyURL(proxyU)
			}
		}
	}

	p.httpClient = &http.Client{
		Timeout:   5 * time.Minute,
		Transport: transport,
	}
}

// VersionsResponse represents the response from the versions endpoint.
type VersionsResponse struct {
	Versions []Version `json:"versions"`
}

// Version represents a provider version.
type Version struct {
	Version   string     `json:"version"`
	Protocols []string   `json:"protocols"`
	Platforms []Platform `json:"platforms"`
}

// Platform represents an OS/arch combination.
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// DownloadInfo represents provider download information.
type DownloadInfo struct {
	Protocols           []string    `json:"protocols"`
	OS                  string      `json:"os"`
	Arch                string      `json:"arch"`
	Filename            string      `json:"filename"`
	DownloadURL         string      `json:"download_url"`
	SHA256Sum           string      `json:"shasum"`
	SHA256SumsURL       string      `json:"shasums_url"`
	SHA256SumsSignature string      `json:"shasums_signature_url"`
	SigningKeys         SigningKeys `json:"signing_keys"`
}

// SigningKeys contains GPG signing information.
type SigningKeys struct {
	GPGPublicKeys []GPGPublicKey `json:"gpg_public_keys"`
}

// GPGPublicKey represents a GPG public key.
type GPGPublicKey struct {
	KeyID          string `json:"key_id"`
	ASCIIArmor     string `json:"ascii_armor"`
	TrustSignature string `json:"trust_signature"`
	Source         string `json:"source"`
	SourceURL      string `json:"source_url"`
}

// GetProviderVersions fetches available versions from upstream registry.
func (p *ProxyService) GetProviderVersions(namespace, name string) (*VersionsResponse, error) {
	url := fmt.Sprintf("%s/v1/providers/%s/%s/versions", p.upstreamURL, namespace, name)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	var versions VersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &versions, nil
}

// GetProviderDownloadInfo fetches download information for a specific provider version.
func (p *ProxyService) GetProviderDownloadInfo(namespace, name, version, osType, arch string) (*DownloadInfo, error) {
	url := fmt.Sprintf("%s/v1/providers/%s/%s/%s/download/%s/%s",
		p.upstreamURL, namespace, name, version, osType, arch)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch download info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	var info DownloadInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// DownloadAndCacheProvider downloads a provider from upstream and caches it locally.
func (p *ProxyService) DownloadAndCacheProvider(namespace, name, version, osType, arch string) (string, string, error) {
	// Build safe directory path with validation
	dirPath, err := buildSafeProviderPath(p.storagePath, namespace, name, version, osType, arch)
	if err != nil {
		return "", "", err
	}

	// Get download info
	info, err := p.GetProviderDownloadInfo(namespace, name, version, osType, arch)
	if err != nil {
		return "", "", err
	}

	// Create storage directory
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Sanitize filename and build file path
	safeFilename, err := sanitizeFilename(info.Filename)
	if err != nil {
		return "", "", err
	}
	filePath := filepath.Join(dirPath, safeFilename)
	if _, err := os.Stat(filePath); err == nil {
		// File exists, verify checksum
		existingSHA256, _ := p.calculateFileSHA256(filePath)
		if existingSHA256 == info.SHA256Sum {
			return filePath, existingSHA256, nil
		}
	}

	// Download the file
	resp, err := p.httpClient.Get(info.DownloadURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to download provider: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file
	tempPath := filePath + ".tmp"
	file, err := os.Create(tempPath) // #nosec G304 - path is constructed from validated components
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write and calculate checksum simultaneously
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		_ = os.Remove(tempPath) // #nosec G104 - best effort cleanup
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}

	calculatedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Verify checksum
	if calculatedSHA256 != info.SHA256Sum {
		_ = os.Remove(tempPath) // #nosec G104 - best effort cleanup
		return "", "", fmt.Errorf("checksum mismatch: expected %s, got %s", info.SHA256Sum, calculatedSHA256)
	}

	// Rename to final path
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath) // #nosec G104 - best effort cleanup
		return "", "", fmt.Errorf("failed to rename file: %w", err)
	}

	return filePath, calculatedSHA256, nil
}

// DownloadAndStoreProvider downloads a provider from a given URL and stores it locally.
// This is used for async caching when the download URL is already known.
func (p *ProxyService) DownloadAndStoreProvider(namespace, name, version, osType, arch, downloadURL string) (string, string, error) {
	// Build safe directory path with validation
	dirPath, err := buildSafeProviderPath(p.storagePath, namespace, name, version, osType, arch)
	if err != nil {
		return "", "", err
	}

	// Create storage directory
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Extract filename from URL and sanitize
	parts := strings.Split(downloadURL, "/")
	filename, err := sanitizeFilename(parts[len(parts)-1])
	if err != nil || filename == "" {
		// Generate safe default filename from validated components
		safeName, _ := sanitizePathComponent(name)
		safeVer, _ := sanitizePathComponent(version)
		safeOS, _ := sanitizePathComponent(osType)
		safeArch, _ := sanitizePathComponent(arch)
		filename = fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", safeName, safeVer, safeOS, safeArch)
	}

	filePath := filepath.Join(dirPath, filename)

	// Check if file already exists
	if existingSHA256, err := p.calculateFileSHA256(filePath); err == nil {
		return filePath, existingSHA256, nil
	}

	// Download the file
	resp, err := p.httpClient.Get(downloadURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to download provider: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file
	tempPath := filePath + ".tmp"
	file, err := os.Create(tempPath) // #nosec G304 - path is constructed from validated components
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write and calculate checksum simultaneously
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		_ = os.Remove(tempPath) // #nosec G104 - best effort cleanup
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}

	calculatedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Rename to final path
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath) // #nosec G104 - best effort cleanup
		return "", "", fmt.Errorf("failed to rename file: %w", err)
	}

	return filePath, calculatedSHA256, nil
}

// SaveUploadedProvider saves an uploaded provider file.
func (p *ProxyService) SaveUploadedProvider(namespace, name, version, osType, arch string, file io.Reader, filename string) (string, string, error) {
	// Build safe directory path with validation
	dirPath, err := buildSafeProviderPath(p.storagePath, namespace, name, version, osType, arch)
	if err != nil {
		return "", "", err
	}

	// Sanitize filename
	safeFilename, err := sanitizeFilename(filename)
	if err != nil {
		return "", "", err
	}

	// Create storage directory
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(dirPath, safeFilename)

	// Create file
	// #nosec G304 -- filePath is constructed from validated components via buildSafeProviderPath and sanitizeFilename
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	// Write and calculate checksum
	hasher := sha256.New()
	writer := io.MultiWriter(outFile, hasher)

	if _, err := io.Copy(writer, file); err != nil {
		_ = os.Remove(filePath) // #nosec G104 - best effort cleanup
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}

	sha256sum := hex.EncodeToString(hasher.Sum(nil))
	return filePath, sha256sum, nil
}

// calculateFileSHA256 calculates the SHA256 checksum of a file.
func (p *ProxyService) calculateFileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath) // #nosec G304 - path is from internal storage
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GetCachedFilePath returns the path to a cached provider file if it exists.
func (p *ProxyService) GetCachedFilePath(namespace, name, version, osType, arch string) (string, bool) {
	// Build safe directory path with validation
	dirPath, err := buildSafeProviderPath(p.storagePath, namespace, name, version, osType, arch)
	if err != nil {
		return "", false
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			// entry.Name() is safe as it comes from filesystem, not user input
			return filepath.Join(dirPath, entry.Name()), true
		}
	}

	return "", false
}

// IsCached checks if a provider is already cached.
func (p *ProxyService) IsCached(namespace, name, version, osType, arch string) bool {
	_, cached := p.GetCachedFilePath(namespace, name, version, osType, arch)
	return cached
}

// SearchResult represents a provider search result.
type SearchResult struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Downloads   int64  `json:"downloads"`
	Source      string `json:"source"`
}

// SearchResponse represents the response from upstream search.
type SearchResponse struct {
	Providers []SearchResult `json:"providers"`
	Total     int            `json:"total"`
}

// SearchProviders searches for providers in upstream registry.
func (p *ProxyService) SearchProviders(query string, limit int) (*SearchResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	result := &SearchResponse{
		Providers: make([]SearchResult, 0),
		Total:     0,
	}
	seen := make(map[string]bool)

	// First, try to find official hashicorp provider with exact name match
	hashicorpURL := fmt.Sprintf("%s/v2/providers?filter[namespace]=hashicorp&filter[name]=%s&page[size]=1",
		p.upstreamURL, url.QueryEscape(query))
	if hashicorpResults, err := p.fetchSearchResults(hashicorpURL); err == nil {
		for _, r := range hashicorpResults {
			key := r.Namespace + "/" + r.Name
			if !seen[key] {
				seen[key] = true
				result.Providers = append(result.Providers, r)
			}
		}
	}

	// Then search by name across all namespaces
	searchURL := fmt.Sprintf("%s/v2/providers?filter[name]=%s&page[size]=%d",
		p.upstreamURL, url.QueryEscape(query), limit)
	if searchResults, err := p.fetchSearchResults(searchURL); err == nil {
		for _, r := range searchResults {
			key := r.Namespace + "/" + r.Name
			if !seen[key] && len(result.Providers) < limit {
				seen[key] = true
				result.Providers = append(result.Providers, r)
			}
		}
	}

	result.Total = len(result.Providers)
	return result, nil
}

// fetchSearchResults fetches search results from a URL.
func (p *ProxyService) fetchSearchResults(searchURL string) ([]SearchResult, error) {
	resp, err := p.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search providers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	// Parse the v2 API response
	var v2Response struct {
		Data []struct {
			Attributes struct {
				Name        string `json:"name"`
				Namespace   string `json:"namespace"`
				Description string `json:"description"`
				Downloads   int64  `json:"downloads"`
				Source      string `json:"source"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&v2Response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(v2Response.Data))
	for _, item := range v2Response.Data {
		results = append(results, SearchResult{
			Namespace:   item.Attributes.Namespace,
			Name:        item.Attributes.Name,
			Description: item.Attributes.Description,
			Downloads:   item.Attributes.Downloads,
			Source:      item.Attributes.Source,
		})
	}

	return results, nil
}
