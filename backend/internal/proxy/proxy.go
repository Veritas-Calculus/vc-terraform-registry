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
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// UpstreamRegistry is the default Terraform registry URL.
const UpstreamRegistry = "https://registry.terraform.io"

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	// Get download info
	info, err := p.GetProviderDownloadInfo(namespace, name, version, osType, arch)
	if err != nil {
		return "", "", err
	}

	// Create storage directory
	dirPath := filepath.Join(p.storagePath, namespace, name, version, osType, arch)
	if err := os.MkdirAll(dirPath, 0750); err != nil { // #nosec G301 - storage directory needs group access
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file already exists
	filePath := filepath.Join(dirPath, info.Filename)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file
	tempPath := filePath + ".tmp"
	file, err := os.Create(tempPath) // #nosec G304 - path is constructed from validated components
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

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
	// Create storage directory
	dirPath := filepath.Join(p.storagePath, namespace, name, version, osType, arch)
	if err := os.MkdirAll(dirPath, 0750); err != nil { // #nosec G301 - storage directory needs group access
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Extract filename from URL
	parts := strings.Split(downloadURL, "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", name, version, osType, arch)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file
	tempPath := filePath + ".tmp"
	file, err := os.Create(tempPath) // #nosec G304 - path is constructed from validated components
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

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
	// Create storage directory
	dirPath := filepath.Join(p.storagePath, namespace, name, version, osType, arch)
	if err := os.MkdirAll(dirPath, 0750); err != nil { // #nosec G301 - storage directory needs group access
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(dirPath, filename)

	// Create file
	outFile, err := os.Create(filePath) // #nosec G304 - path is constructed from validated components
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

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
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GetCachedFilePath returns the path to a cached provider file if it exists.
func (p *ProxyService) GetCachedFilePath(namespace, name, version, osType, arch string) (string, bool) {
	dirPath := filepath.Join(p.storagePath, namespace, name, version, osType, arch)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
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
	defer resp.Body.Close()

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
