// Package api provides HTTP handlers for the Provider Mirror Protocol.
package api

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/proxy"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// logSanitizer removes control characters that could be used for log injection attacks.
// This includes newlines, carriage returns, tabs, and other control characters.
var logSanitizer = regexp.MustCompile(`[\x00-\x1f\x7f]`)

// logSafeChars matches any character that is NOT in a conservative "safe for logs" set.
// Allowed characters are alphanumerics, space, and a limited set of punctuation.
// Everything else is replaced with '?' to avoid confusing or forging log structure.
var logSafeChars = regexp.MustCompile(`[^a-zA-Z0-9 .,_:@/\-]`)

// validIdentifier validates Terraform provider identifiers (namespace, name).
// Valid identifiers contain only lowercase alphanumerics, hyphens, and underscores.
// This validation prevents log injection by rejecting malicious input at the entry point.
var validIdentifier = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// validVersion validates semantic version strings.
// Allows formats like: 1.0.0, 1.0.0-beta, 1.0.0-rc.1, 1.0.0+build
var validVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9._-]+)?(\+[a-zA-Z0-9._-]+)?$`)

// validateProviderParams validates namespace, name, and version parameters.
// Returns an error message if validation fails, or empty string if valid.
// This is the primary defense against log injection - validated inputs are safe to log.
func validateProviderParams(namespace, name, version string) string {
	if len(namespace) > 64 || !validIdentifier.MatchString(namespace) {
		return "invalid namespace: must be 1-64 lowercase alphanumeric characters, hyphens, or underscores"
	}
	if len(name) > 64 || !validIdentifier.MatchString(name) {
		return "invalid name: must be 1-64 lowercase alphanumeric characters, hyphens, or underscores"
	}
	if version != "" {
		if len(version) > 64 || !validVersion.MatchString(version) {
			return "invalid version: must be a valid semantic version (e.g., 1.0.0)"
		}
	}
	return ""
}

// sanitizeForLog removes potentially dangerous characters from user input for safe logging.
// This prevents log injection attacks by removing all control characters and restricting
// the remaining characters to a conservative safe set.
func sanitizeForLog(s string) string {
	// First, remove all control characters (including newlines, tabs, etc.).
	safe := logSanitizer.ReplaceAllString(s, "")

	// Replace any character outside the allowed safe set with '?'.
	safe = logSafeChars.ReplaceAllString(safe, "?")

	// Trim leading and trailing spaces to avoid confusing log formatting.
	safe = strings.TrimSpace(safe)

	// Limit length to prevent log flooding.
	if len(safe) > 100 {
		safe = safe[:100] + "..."
	}

	return safe
}

// safeLogf is a helper that sanitizes all arguments before logging.
// This provides defense-in-depth against log injection.
func safeLogf(format string, args ...interface{}) {
	sanitizedArgs := make([]interface{}, len(args))
	for i, arg := range args {
		if arg == nil {
			sanitizedArgs[i] = nil
			continue
		}

		// Obtain a string representation of the argument.
		var s string
		if str, ok := arg.(string); ok {
			s = str
		} else {
			s = fmt.Sprint(arg)
		}

		// Sanitize and quote to make any remaining special characters visible.
		safe := sanitizeForLog(s)
		sanitizedArgs[i] = strconv.Quote(safe)
	}

	// Format the message first, then log it as a single safe string.
	msg := fmt.Sprintf(format, sanitizedArgs...)
	log.Print(msg) //nolint:govet // format string is always a compile-time constant from caller
}

// ProviderMirrorHandler handles Terraform Provider Mirror Protocol requests.
// This implements the protocol defined at:
// https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol
type ProviderMirrorHandler struct {
	db           *gorm.DB
	storagePath  string
	proxyService *proxy.ProxyService
}

// NewProviderMirrorHandler creates a new ProviderMirrorHandler instance.
func NewProviderMirrorHandler(db *gorm.DB, storagePath string) *ProviderMirrorHandler {
	h := &ProviderMirrorHandler{
		db:           db,
		storagePath:  storagePath,
		proxyService: proxy.NewProxyService(storagePath, ""),
	}
	h.refreshProxySettings()
	return h
}

// refreshProxySettings loads proxy settings from database and updates the proxy service.
func (h *ProviderMirrorHandler) refreshProxySettings() {
	var settings models.Settings
	if err := h.db.First(&settings).Error; err == nil {
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}
}

// ArchiveInfo represents archive information for a platform.
type ArchiveInfo struct {
	URL    string   `json:"url"`
	Hashes []string `json:"hashes,omitempty"`
}

// getHostAndScheme extracts host and scheme from request headers.
func getHostAndScheme(c *gin.Context) (string, string) {
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}

	scheme := "https"
	if forwardedProto := c.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	} else if c.Request.TLS == nil {
		scheme = "http"
	}

	return host, scheme
}

// ListAvailableVersions returns the index.json for a provider.
// Path: /{hostname}/{namespace}/{name}/index.json
// Always queries upstream (if allowed) and merges with local versions.
func (h *ProviderMirrorHandler) ListAvailableVersions(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Validate input parameters to prevent log injection and ensure data integrity.
	if errMsg := validateProviderParams(namespace, name, ""); errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	// Query all versions for this provider from the database
	var providers []models.Provider
	if err := h.db.Where("namespace = ? AND name = ?", namespace, name).
		Order("version DESC").
		Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build the versions map from local providers
	versions := make(map[string]interface{})
	for _, p := range providers {
		versions[p.Version] = struct{}{}
	}

	// Check if online search is allowed
	var settings models.Settings
	allowOnline := true
	if err := h.db.First(&settings).Error; err == nil {
		allowOnline = settings.AllowOnlineSearch
		// Refresh proxy settings
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}

	// Always try to get upstream versions if online search is allowed
	if allowOnline {
		upstreamVersions, err := h.proxyService.GetProviderVersions(namespace, name)
		if err == nil {
			// Add upstream versions to the response
			for _, v := range upstreamVersions.Versions {
				versions[v.Version] = struct{}{}
			}
		}
	}

	// If still no versions, return 404
	if len(versions) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "provider not found",
		})
		return
	}

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
	})
}

// GetVersionArchives returns the version.json for a specific provider version.
// Path: /{hostname}/{namespace}/{name}/{version}.json
// Always returns all platforms from upstream, using local cache info when available.
func (h *ProviderMirrorHandler) GetVersionArchives(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Param("version")

	// Remove .json suffix if present
	version = strings.TrimSuffix(version, ".json")

	// Validate input parameters to prevent log injection and ensure data integrity.
	// After validation, these values are safe to use in logs without sanitization.
	if errMsg := validateProviderParams(namespace, name, version); errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	host, scheme := getHostAndScheme(c)

	// Check if online search is allowed and refresh proxy settings
	var settings models.Settings
	allowOnline := true
	if err := h.db.First(&settings).Error; err == nil {
		allowOnline = settings.AllowOnlineSearch
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}

	// Query local platforms for this provider version
	var localPlatforms []models.ProviderPlatform
	h.db.Joins("JOIN providers ON providers.id = provider_platforms.provider_id").
		Where("providers.namespace = ? AND providers.name = ? AND providers.version = ?",
			namespace, name, version).
		Find(&localPlatforms)

	// Build a map of local platforms for quick lookup
	localPlatformMap := make(map[string]models.ProviderPlatform)
	for _, p := range localPlatforms {
		key := p.OS + "_" + p.Arch
		localPlatformMap[key] = p
	}

	// If online search is allowed, get upstream platforms to ensure we have complete list
	var upstreamPlatforms []proxy.Platform
	if allowOnline {
		upstreamVersions, err := h.proxyService.GetProviderVersions(namespace, name)
		if err == nil {
			for _, v := range upstreamVersions.Versions {
				if v.Version == version {
					upstreamPlatforms = v.Platforms
					break
				}
			}
		}
	}

	// If no upstream platforms and no local platforms, return 404
	if len(upstreamPlatforms) == 0 && len(localPlatforms) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	// Build archives map
	archives := make(map[string]ArchiveInfo)

	// If we have upstream platforms, use them as the complete list
	if len(upstreamPlatforms) > 0 {
		for _, p := range upstreamPlatforms {
			key := p.OS + "_" + p.Arch
			downloadURL := scheme + "://" + host + "/v1/providers/" + namespace + "/" + name + "/" + version + "/download/" + p.OS + "/" + p.Arch + "/binary"

			archive := ArchiveInfo{
				URL: downloadURL,
			}

			// If we have local cache, add the hash
			if localP, ok := localPlatformMap[key]; ok && localP.SHA256Sum != "" {
				archive.Hashes = []string{"zh:" + localP.SHA256Sum}
			}

			archives[key] = archive
		}

		// Trigger async caching if we don't have all platforms locally
		if len(localPlatforms) < len(upstreamPlatforms) {
			go h.asyncCacheProvider(namespace, name, version, upstreamPlatforms)
		}
	} else {
		// No upstream, use local platforms only
		archives = h.buildLocalArchives(localPlatforms, namespace, name, version, host, scheme)
	}

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{"archives": archives})
}

// buildLocalArchives creates archive info from local platforms.
func (h *ProviderMirrorHandler) buildLocalArchives(platforms []models.ProviderPlatform, namespace, name, version, host, scheme string) map[string]ArchiveInfo {
	archives := make(map[string]ArchiveInfo)

	for _, p := range platforms {
		key := p.OS + "_" + p.Arch
		downloadURL := scheme + "://" + host + "/v1/providers/" + namespace + "/" + name + "/" + version + "/download/" + p.OS + "/" + p.Arch + "/binary"

		archive := ArchiveInfo{
			URL: downloadURL,
		}

		if p.SHA256Sum != "" {
			archive.Hashes = []string{"zh:" + p.SHA256Sum}
		}

		archives[key] = archive
	}

	return archives
}

// asyncCacheProvider downloads and caches a provider version in the background.
// IMPORTANT: This function assumes that namespace, name, and version have already been
// validated by validateProviderParams() before being passed here. Validated inputs contain
// only safe characters (alphanumerics, hyphens, underscores, dots) and are safe to log directly.
func (h *ProviderMirrorHandler) asyncCacheProvider(namespace, name, version string, platforms []proxy.Platform) {
	// Inputs are pre-validated by the caller, so they are safe to log directly.
	log.Printf("[AsyncCache] Starting background cache for %s/%s v%s (%d platforms)",
		namespace, name, version, len(platforms))

	// Refresh proxy settings
	h.refreshProxySettings()

	// Check if already cached (in case of race condition)
	var existingProvider models.Provider
	if err := h.db.Where("namespace = ? AND name = ? AND version = ?", namespace, name, version).
		First(&existingProvider).Error; err == nil {
		log.Printf("[AsyncCache] Provider %s/%s v%s already cached, skipping", namespace, name, version)
		return
	}

	// Create provider record
	provider := models.Provider{
		Namespace:   namespace,
		Name:        name,
		Version:     version,
		Description: "Async cached from upstream",
		SourceType:  models.SourceMirror,
		SourceURL:   "https://registry.terraform.io",
		Protocols:   `["5.0", "6.0"]`,
		Published:   time.Now(),
	}

	if err := h.db.Create(&provider).Error; err != nil {
		log.Printf("[AsyncCache] Failed to create provider record: %v", err)
		return
	}

	successCount := 0
	for _, p := range platforms {
		// Download and store each platform binary
		downloadInfo, err := h.proxyService.GetProviderDownloadInfo(namespace, name, version, p.OS, p.Arch)
		if err != nil {
			// p.OS and p.Arch come from upstream API, use safeLogf for safety
			safeLogf("[AsyncCache] Failed to get download info for %s/%s: %v", p.OS, p.Arch, err)
			continue
		}

		filePath, sha256sum, err := h.proxyService.DownloadAndStoreProvider(
			namespace, name, version, p.OS, p.Arch, downloadInfo.DownloadURL,
		)
		if err != nil {
			safeLogf("[AsyncCache] Failed to download %s/%s: %v", p.OS, p.Arch, err)
			continue
		}

		// Create platform record
		platform := models.ProviderPlatform{
			ProviderID: provider.ID,
			OS:         p.OS,
			Arch:       p.Arch,
			Filename:   downloadInfo.Filename,
			FilePath:   filePath,
			SHA256Sum:  sha256sum,
		}

		if err := h.db.Create(&platform).Error; err != nil {
			log.Printf("[AsyncCache] Failed to create platform record: %v", err)
			continue
		}

		successCount++
		// p.OS and p.Arch come from upstream API, use safeLogf for safety
		safeLogf("[AsyncCache] Cached %s/%s v%s %s_%s", namespace, name, version, p.OS, p.Arch)
	}

	log.Printf("[AsyncCache] Completed caching %s/%s v%s: %d/%d platforms successful",
		namespace, name, version, successCount, len(platforms))
}
