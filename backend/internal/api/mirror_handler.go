// Package api provides HTTP handlers for the API.
package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/proxy"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MirrorHandler handles provider mirroring operations.
type MirrorHandler struct {
	db           *gorm.DB
	proxyService *proxy.ProxyService
	storagePath  string
}

// NewMirrorHandler creates a new MirrorHandler instance.
func NewMirrorHandler(db *gorm.DB, storagePath string) *MirrorHandler {
	h := &MirrorHandler{
		db:           db,
		proxyService: proxy.NewProxyService(storagePath, ""),
		storagePath:  storagePath,
	}
	h.refreshProxySettings()
	return h
}

// refreshProxySettings loads proxy settings from database and updates the proxy service.
func (h *MirrorHandler) refreshProxySettings() {
	var settings models.Settings
	if err := h.db.First(&settings).Error; err == nil {
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}
}

// platformInfo holds OS and Arch for a platform.
type platformInfo struct {
	OS   string
	Arch string
}

// getPlatformsForVersion extracts matching platforms from version info.
func getPlatformsForVersion(versions *proxy.VersionsResponse, version, osType, arch string) ([]platformInfo, string) {
	if version == "" && len(versions.Versions) > 0 {
		version = versions.Versions[0].Version
	}

	var platforms []platformInfo
	for _, v := range versions.Versions {
		if v.Version == version {
			for _, p := range v.Platforms {
				if (osType == "all" || osType == p.OS) && (arch == "all" || arch == p.Arch) {
					platforms = append(platforms, platformInfo{OS: p.OS, Arch: p.Arch})
				}
			}
			break
		}
	}
	return platforms, version
}

// getOrCreateProvider retrieves or creates a provider in the database.
func (h *MirrorHandler) getOrCreateProvider(namespace, name, version, protocols string) (*models.Provider, error) {
	var provider models.Provider
	result := h.db.Where("namespace = ? AND name = ? AND version = ?", namespace, name, version).First(&provider)

	if result.Error == gorm.ErrRecordNotFound {
		if protocols == "" {
			protocols = `["5.0"]`
		}
		provider = models.Provider{
			Namespace:  namespace,
			Name:       name,
			Version:    version,
			SourceType: models.SourceMirror,
			SourceURL:  "https://registry.terraform.io",
			Published:  time.Now(),
			Protocols:  protocols,
		}
		if err := h.db.Create(&provider).Error; err != nil {
			return nil, err
		}
	}
	return &provider, nil
}

// savePlatformEntry creates or updates a platform entry in the database.
func (h *MirrorHandler) savePlatformEntry(providerID uint, plat models.ProviderPlatform) {
	var existingPlatform models.ProviderPlatform
	if err := h.db.Where("provider_id = ? AND os = ? AND arch = ?",
		providerID, plat.OS, plat.Arch).First(&existingPlatform).Error; err == nil {
		existingPlatform.FilePath = plat.FilePath
		existingPlatform.Filename = plat.Filename
		existingPlatform.SHA256Sum = plat.SHA256Sum
		existingPlatform.FileSize = plat.FileSize
		h.db.Save(&existingPlatform)
	} else {
		plat.ProviderID = providerID
		h.db.Create(&plat)
	}
}

// UploadProvider handles manual provider upload.
func (h *MirrorHandler) UploadProvider(c *gin.Context) {
	namespace := c.PostForm("namespace")
	name := c.PostForm("name")
	version := c.PostForm("version")
	osType := c.PostForm("os")
	arch := c.PostForm("arch")
	description := c.PostForm("description")

	if namespace == "" || name == "" || version == "" || osType == "" || arch == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "namespace, name, version, os, and arch are required",
		})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() { _ = file.Close() }()

	// Save the file
	filePath, sha256sum, err := h.proxyService.SaveUploadedProvider(
		namespace, name, version, osType, arch, file, header.Filename,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create or update provider in database
	var provider models.Provider
	result := h.db.Where("namespace = ? AND name = ? AND version = ?",
		namespace, name, version).First(&provider)

	if result.Error == gorm.ErrRecordNotFound {
		provider = models.Provider{
			Namespace:   namespace,
			Name:        name,
			Version:     version,
			Description: description,
			SourceType:  models.SourceUpload,
			Published:   time.Now(),
			Protocols:   `["5.0"]`,
		}
		if err := h.db.Create(&provider).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Create platform entry
	platform := models.ProviderPlatform{
		ProviderID: provider.ID,
		OS:         osType,
		Arch:       arch,
		Filename:   header.Filename,
		FilePath:   filePath,
		SHA256Sum:  sha256sum,
		FileSize:   header.Size,
	}

	// Check if platform already exists
	var existingPlatform models.ProviderPlatform
	if err := h.db.Where("provider_id = ? AND os = ? AND arch = ?",
		provider.ID, osType, arch).First(&existingPlatform).Error; err == nil {
		// Update existing
		existingPlatform.Filename = header.Filename
		existingPlatform.FilePath = filePath
		existingPlatform.SHA256Sum = sha256sum
		existingPlatform.FileSize = header.Size
		h.db.Save(&existingPlatform)
	} else {
		h.db.Create(&platform)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Provider uploaded successfully",
		"provider":  provider,
		"platform":  platform,
		"sha256sum": sha256sum,
	})
}

// MirrorProgress represents a progress update for SSE.
type MirrorProgress struct {
	Type           string  `json:"type"`             // "progress", "complete", "error"
	Current        int     `json:"current"`          // Current platform index (1-based)
	Total          int     `json:"total"`            // Total platforms to download
	Platform       string  `json:"platform"`         // Current platform being downloaded
	Percent        float64 `json:"percent"`          // Overall percentage
	BytesPerSecond int64   `json:"bytes_per_second"` // Download speed
	ETASeconds     float64 `json:"eta_seconds"`      // Estimated time remaining
	Message        string  `json:"message"`          // Status message
	Error          string  `json:"error,omitempty"`  // Error message if any
}

// MirrorProviderWithProgress mirrors a provider with SSE progress updates.
func (h *MirrorHandler) MirrorProviderWithProgress(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Query("version")
	osType := c.DefaultQuery("os", "all")
	arch := c.DefaultQuery("arch", "all")
	proxyURL := c.Query("proxy_url")

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	sendProgress := func(p MirrorProgress) {
		data, _ := json.Marshal(p)
		c.SSEvent("message", string(data))
		c.Writer.Flush()
	}

	if namespace == "" || name == "" {
		sendProgress(MirrorProgress{Type: "error", Error: "namespace and name are required"})
		return
	}

	// Create proxy service
	proxyService := h.getProxyService(proxyURL)
	if proxyURL != "" {
		sendProgress(MirrorProgress{Type: "progress", Message: fmt.Sprintf("Using proxy: %s", proxyURL)})
	}

	// Get platforms to mirror
	sendProgress(MirrorProgress{Type: "progress", Message: "Fetching version information..."})
	platforms, resolvedVersion, err := h.fetchPlatformsToMirror(proxyService, namespace, name, version, osType, arch)
	if err != nil {
		sendProgress(MirrorProgress{Type: "error", Error: err.Error()})
		return
	}
	version = resolvedVersion

	total := len(platforms)
	sendProgress(MirrorProgress{Type: "progress", Total: total, Message: fmt.Sprintf("Found %d platforms to mirror", total)})

	// Download all platforms with progress
	mirroredPlatforms, totalBytes, lastError := h.downloadPlatformsWithProgress(proxyService, namespace, name, version, platforms, sendProgress)

	if len(mirroredPlatforms) == 0 {
		sendProgress(MirrorProgress{Type: "error", Error: fmt.Sprintf("Failed to mirror any platform: %v", lastError)})
		return
	}

	// Save to database
	if err := h.saveMirroredProvider(proxyService, namespace, name, version, mirroredPlatforms); err != nil {
		sendProgress(MirrorProgress{Type: "error", Error: err.Error()})
		return
	}

	sendProgress(MirrorProgress{
		Type:    "complete",
		Current: total,
		Total:   total,
		Percent: 100,
		Message: fmt.Sprintf("Successfully mirrored %d platforms (%.2f MB total)", len(mirroredPlatforms), float64(totalBytes)/1024/1024),
	})
}

// getProxyService returns the appropriate proxy service based on proxyURL.
func (h *MirrorHandler) getProxyService(proxyURL string) *proxy.ProxyService {
	if proxyURL != "" {
		return proxy.NewProxyServiceWithProxy(h.storagePath, "", proxyURL, "http")
	}
	return h.proxyService
}

// fetchPlatformsToMirror fetches version info and returns platforms to download.
func (h *MirrorHandler) fetchPlatformsToMirror(proxyService *proxy.ProxyService, namespace, name, version, osType, arch string) ([]platformInfo, string, error) {
	versions, err := proxyService.GetProviderVersions(namespace, name)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get versions: %v", err)
	}
	if len(versions.Versions) == 0 {
		return nil, "", fmt.Errorf("no versions available")
	}

	platforms, resolvedVersion := getPlatformsForVersion(versions, version, osType, arch)
	if len(platforms) == 0 {
		return nil, "", fmt.Errorf("no matching platforms found")
	}
	return platforms, resolvedVersion, nil
}

// downloadPlatformsWithProgress downloads platforms and sends progress updates.
func (h *MirrorHandler) downloadPlatformsWithProgress(proxyService *proxy.ProxyService, namespace, name, version string, platforms []platformInfo, sendProgress func(MirrorProgress)) ([]models.ProviderPlatform, int64, error) {
	var mirroredPlatforms []models.ProviderPlatform
	var lastError error
	var totalBytes int64
	total := len(platforms)
	startTime := time.Now()

	for i, plat := range platforms {
		platformStr := fmt.Sprintf("%s/%s", plat.OS, plat.Arch)
		sendProgress(MirrorProgress{
			Type: "progress", Current: i + 1, Total: total, Platform: platformStr,
			Percent: float64(i) / float64(total) * 100, Message: fmt.Sprintf("Downloading %s...", platformStr),
		})

		platStart := time.Now()
		filePath, sha256sum, err := proxyService.DownloadAndCacheProvider(namespace, name, version, plat.OS, plat.Arch)
		if err != nil {
			lastError = err
			sendProgress(MirrorProgress{Type: "progress", Current: i + 1, Total: total, Platform: platformStr, Message: fmt.Sprintf("Failed: %v", err)})
			continue
		}

		fileSize := getFileSize(filePath)
		totalBytes += fileSize
		platSpeed := int64(float64(fileSize) / time.Since(platStart).Seconds())
		etaSeconds := calculateETA(totalBytes, i+1, total, time.Since(startTime).Seconds())

		sendProgress(MirrorProgress{
			Type: "progress", Current: i + 1, Total: total, Platform: platformStr,
			Percent: float64(i+1) / float64(total) * 100, BytesPerSecond: platSpeed, ETASeconds: etaSeconds,
			Message: fmt.Sprintf("Downloaded %s (%.2f MB)", platformStr, float64(fileSize)/1024/1024),
		})

		mirroredPlatforms = append(mirroredPlatforms, models.ProviderPlatform{
			OS: plat.OS, Arch: plat.Arch, Filename: filepath.Base(filePath),
			FilePath: filePath, SHA256Sum: sha256sum, FileSize: fileSize,
		})
	}
	return mirroredPlatforms, totalBytes, lastError
}

// getFileSize returns the size of a file, or 0 if it cannot be determined.
func getFileSize(filePath string) int64 {
	if info, err := os.Stat(filePath); err == nil {
		return info.Size()
	}
	return 0
}

// calculateETA estimates remaining time based on progress.
func calculateETA(totalBytes int64, completed, total int, elapsed float64) float64 {
	if elapsed <= 0 || completed <= 0 || completed >= total {
		return 0
	}
	bytesPerSecond := float64(totalBytes) / elapsed
	avgBytesPerPlatform := float64(totalBytes) / float64(completed)
	remaining := total - completed
	return avgBytesPerPlatform * float64(remaining) / bytesPerSecond
}

// saveMirroredProvider saves the provider and platforms to the database.
func (h *MirrorHandler) saveMirroredProvider(proxyService *proxy.ProxyService, namespace, name, version string, platforms []models.ProviderPlatform) error {
	protocols := `["5.0"]`
	if len(platforms) > 0 {
		if info, err := proxyService.GetProviderDownloadInfo(namespace, name, version, platforms[0].OS, platforms[0].Arch); err == nil && len(info.Protocols) > 0 {
			protocols = fmt.Sprintf(`["%s"]`, info.Protocols[0])
		}
	}

	provider, err := h.getOrCreateProvider(namespace, name, version, protocols)
	if err != nil {
		return err
	}

	for _, plat := range platforms {
		h.savePlatformEntry(provider.ID, plat)
	}
	return nil
}

// MirrorProvider mirrors a provider from upstream registry (non-SSE version for backwards compatibility).
func (h *MirrorHandler) MirrorProvider(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Query("version")
	osType := c.DefaultQuery("os", "all")
	arch := c.DefaultQuery("arch", "all")

	if namespace == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and name are required"})
		return
	}

	// Fetch platforms to mirror
	platforms, resolvedVersion, err := h.fetchPlatformsToMirror(h.proxyService, namespace, name, version, osType, arch)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	version = resolvedVersion

	// Download all platforms
	mirroredPlatforms, lastError := h.downloadPlatforms(h.proxyService, namespace, name, version, platforms)

	if len(mirroredPlatforms) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to mirror any platform: %v", lastError)})
		return
	}

	// Save to database
	if err := h.saveMirroredProvider(h.proxyService, namespace, name, version, mirroredPlatforms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Load saved provider for response
	var provider models.Provider
	h.db.Where("namespace = ? AND name = ? AND version = ?", namespace, name, version).Preload("Platforms").First(&provider)

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Provider mirrored successfully (%d platforms)", len(mirroredPlatforms)),
		"provider":  provider,
		"platforms": provider.Platforms,
	})
}

// downloadPlatforms downloads all specified platforms without progress updates.
func (h *MirrorHandler) downloadPlatforms(proxyService *proxy.ProxyService, namespace, name, version string, platforms []platformInfo) ([]models.ProviderPlatform, error) {
	var mirroredPlatforms []models.ProviderPlatform
	var lastError error

	for _, plat := range platforms {
		filePath, sha256sum, err := proxyService.DownloadAndCacheProvider(namespace, name, version, plat.OS, plat.Arch)
		if err != nil {
			lastError = err
			continue
		}

		mirroredPlatforms = append(mirroredPlatforms, models.ProviderPlatform{
			OS: plat.OS, Arch: plat.Arch, Filename: filepath.Base(filePath),
			FilePath: filePath, SHA256Sum: sha256sum, FileSize: getFileSize(filePath),
		})
	}
	return mirroredPlatforms, lastError
}

// ListUpstreamVersions lists available versions from upstream.
func (h *MirrorHandler) ListUpstreamVersions(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Check if online search is allowed and update proxy settings
	var settings models.Settings
	if err := h.db.First(&settings).Error; err == nil {
		if !settings.AllowOnlineSearch {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Online search is disabled",
			})
			return
		}
		// Update proxy settings before making upstream request
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}

	versions, err := h.proxyService.GetProviderVersions(namespace, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Failed to get versions: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// DownloadProvider handles provider binary downloads.
// If the provider is not cached locally, it will download from upstream,
// cache it, and serve it to the client.
func (h *MirrorHandler) DownloadProvider(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Param("version")
	osType := c.Param("os")
	arch := c.Param("arch")

	// Check if online search is allowed for auto-mirroring and update proxy settings
	var settings models.Settings
	allowOnline := true
	if err := h.db.First(&settings).Error; err == nil {
		allowOnline = settings.AllowOnlineSearch
		// Update proxy settings before making upstream request
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}

	// Find the provider and platform
	var provider models.Provider
	if err := h.db.Where("namespace = ? AND name = ? AND version = ?",
		namespace, name, version).First(&provider).Error; err != nil {
		// Provider not found locally, try to download from upstream
		if !allowOnline {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		h.downloadAndCacheFromUpstream(c, namespace, name, version, osType, arch)
		return
	}

	var platform models.ProviderPlatform
	if err := h.db.Where("provider_id = ? AND os = ? AND arch = ?",
		provider.ID, osType, arch).First(&platform).Error; err != nil {
		// Platform not found locally, try to download from upstream
		if !allowOnline {
			c.JSON(http.StatusNotFound, gin.H{"error": "Platform not found"})
			return
		}
		h.downloadAndCacheFromUpstream(c, namespace, name, version, osType, arch)
		return
	}

	// Increment download counter
	h.db.Model(&provider).Update("downloads", gorm.Expr("downloads + 1"))

	// Serve the file
	c.File(platform.FilePath)
}

// downloadAndCacheFromUpstream downloads a provider from upstream, caches it, and serves it.
func (h *MirrorHandler) downloadAndCacheFromUpstream(c *gin.Context, namespace, name, version, osType, arch string) {
	// Get download info from upstream
	downloadInfo, err := h.proxyService.GetProviderDownloadInfo(namespace, name, version, osType, arch)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found upstream: " + err.Error()})
		return
	}

	// Download and cache the provider
	filePath, sha256sum, err := h.proxyService.DownloadAndCacheProvider(namespace, name, version, osType, arch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download provider: " + err.Error()})
		return
	}

	// Get file info for size
	fileInfo, _ := os.Stat(filePath)
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	// Create or get provider record
	var provider models.Provider
	if err := h.db.Where("namespace = ? AND name = ? AND version = ?",
		namespace, name, version).First(&provider).Error; err != nil {
		// Create new provider
		provider = models.Provider{
			Namespace:   namespace,
			Name:        name,
			Version:     version,
			Description: "Auto-cached from upstream",
			SourceType:  models.SourceMirror,
			SourceURL:   "https://registry.terraform.io",
			Protocols:   `["5.0", "6.0"]`,
			Published:   time.Now(),
		}
		h.db.Create(&provider)
	}

	// Create platform record if not exists
	var platform models.ProviderPlatform
	if err := h.db.Where("provider_id = ? AND os = ? AND arch = ?",
		provider.ID, osType, arch).First(&platform).Error; err != nil {
		platform = models.ProviderPlatform{
			ProviderID: provider.ID,
			OS:         osType,
			Arch:       arch,
			Filename:   downloadInfo.Filename,
			FilePath:   filePath,
			SHA256Sum:  sha256sum,
			FileSize:   fileSize,
		}
		h.db.Create(&platform)
	}

	// Increment download counter
	h.db.Model(&provider).Update("downloads", gorm.Expr("downloads + 1"))

	// Serve the file
	c.File(filePath)
}

// GetProviderDownloadInfo returns download info following Terraform protocol.
func (h *MirrorHandler) GetProviderDownloadInfo(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Param("version")
	osType := c.Param("os")
	arch := c.Param("arch")

	// Check if online search is allowed and update proxy settings
	var settings models.Settings
	allowOnline := true
	if err := h.db.First(&settings).Error; err == nil {
		allowOnline = settings.AllowOnlineSearch
		// Update proxy settings before making upstream request
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}

	// Check if we have it locally
	var provider models.Provider
	var platform models.ProviderPlatform
	hasLocal := false

	if err := h.db.Where("namespace = ? AND name = ? AND version = ?",
		namespace, name, version).First(&provider).Error; err == nil {
		if err := h.db.Where("provider_id = ? AND os = ? AND arch = ?",
			provider.ID, osType, arch).First(&platform).Error; err == nil {
			hasLocal = true
		}
	}

	// Get the host from request
	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := c.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	if hasLocal {
		// Return local download info
		downloadURL := fmt.Sprintf("%s://%s/v1/providers/%s/%s/%s/download/%s/%s/binary",
			scheme, host, namespace, name, version, osType, arch)

		c.JSON(http.StatusOK, gin.H{
			"protocols":             []string{"5.0"},
			"os":                    osType,
			"arch":                  arch,
			"filename":              platform.Filename,
			"download_url":          downloadURL,
			"shasum":                platform.SHA256Sum,
			"shasums_url":           "",
			"shasums_signature_url": "",
			"signing_keys": gin.H{
				"gpg_public_keys": []gin.H{},
			},
		})
		return
	}

	// Only fetch from upstream if online search is allowed
	if !allowOnline {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Try to get from upstream and cache
	info, err := h.proxyService.GetProviderDownloadInfo(namespace, name, version, osType, arch)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Start background caching
	go func() {
		_, _, _ = h.proxyService.DownloadAndCacheProvider(namespace, name, version, osType, arch) // #nosec G104 - async cache
	}()

	// Return upstream info but with our download URL
	downloadURL := fmt.Sprintf("%s://%s/v1/providers/%s/%s/%s/download/%s/%s/binary",
		scheme, host, namespace, name, version, osType, arch)

	c.JSON(http.StatusOK, gin.H{
		"protocols":             info.Protocols,
		"os":                    osType,
		"arch":                  arch,
		"filename":              info.Filename,
		"download_url":          downloadURL,
		"shasum":                info.SHA256Sum,
		"shasums_url":           info.SHA256SumsURL,
		"shasums_signature_url": info.SHA256SumsSignature,
		"signing_keys":          info.SigningKeys,
	})
}

// GetProviderVersions returns available versions following Terraform protocol.
func (h *MirrorHandler) GetProviderVersions(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Get local versions
	var providers []models.Provider
	h.db.Where("namespace = ? AND name = ?", namespace, name).
		Order("version DESC").Find(&providers)

	// Build versions response
	versions := make([]gin.H, 0)
	versionMap := make(map[string]bool)

	for _, p := range providers {
		if versionMap[p.Version] {
			continue
		}
		versionMap[p.Version] = true

		// Get platforms for this version
		var platforms []models.ProviderPlatform
		h.db.Where("provider_id = ?", p.ID).Find(&platforms)

		platformList := make([]gin.H, 0)
		for _, pl := range platforms {
			platformList = append(platformList, gin.H{
				"os":   pl.OS,
				"arch": pl.Arch,
			})
		}

		versions = append(versions, gin.H{
			"version":   p.Version,
			"protocols": []string{"5.0"},
			"platforms": platformList,
		})
	}

	// If no local versions, try upstream (if allowed)
	if len(versions) == 0 {
		// Check if online search is allowed and update proxy settings
		var settings models.Settings
		allowOnline := true
		if err := h.db.First(&settings).Error; err == nil {
			allowOnline = settings.AllowOnlineSearch
			// Update proxy settings before making upstream request
			h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
		}

		if !allowOnline {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}

		upstreamVersions, err := h.proxyService.GetProviderVersions(namespace, name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}

		for _, v := range upstreamVersions.Versions {
			platformList := make([]gin.H, 0)
			for _, p := range v.Platforms {
				platformList = append(platformList, gin.H{
					"os":   p.OS,
					"arch": p.Arch,
				})
			}
			versions = append(versions, gin.H{
				"version":   v.Version,
				"protocols": v.Protocols,
				"platforms": platformList,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
	})
}

// ListMirroredProviders lists all mirrored providers.
// MirroredProviderSummary represents a mirrored provider with aggregated version info.
type MirroredProviderSummary struct {
	ID            uint   `json:"id"`
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	LatestVersion string `json:"version"`
	VersionCount  int    `json:"version_count"`
	Downloads     int    `json:"downloads"`
	SourceType    string `json:"source_type"`
	Published     string `json:"published"`
	PlatformCount int    `json:"platform_count"`
}

func (h *MirrorHandler) ListMirroredProviders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Query to get unique providers grouped by namespace/name
	var results []struct {
		ID            uint
		Namespace     string
		Name          string
		Description   string
		LatestVersion string
		VersionCount  int64
		Downloads     int64
		SourceType    string
		Published     string
		PlatformCount int64
	}

	baseQuery := h.db.Model(&models.Provider{})

	if sourceType := c.Query("source_type"); sourceType != "" {
		baseQuery = baseQuery.Where("source_type = ?", sourceType)
	}

	// Count unique providers
	var total int64
	h.db.Model(&models.Provider{}).Select("COUNT(DISTINCT namespace || '/' || name)").Scan(&total)

	// Get aggregated provider info with unique platform count
	if err := baseQuery.Select(`
		MAX(providers.id) as id,
		namespace,
		name,
		MAX(description) as description,
		MAX(version) as latest_version,
		COUNT(DISTINCT version) as version_count,
		SUM(downloads) as downloads,
		MAX(source_type) as source_type,
		MAX(published) as published,
		(SELECT COUNT(DISTINCT os || '_' || arch) FROM provider_platforms WHERE provider_id IN 
			(SELECT id FROM providers p2 WHERE p2.namespace = providers.namespace AND p2.name = providers.name)
		) as platform_count
	`).
		Group("namespace, name").
		Order("MAX(updated_at) DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to MirroredProviderSummary
	providers := make([]MirroredProviderSummary, len(results))
	for i, r := range results {
		providers[i] = MirroredProviderSummary{
			ID:            r.ID,
			Namespace:     r.Namespace,
			Name:          r.Name,
			Description:   r.Description,
			LatestVersion: r.LatestVersion,
			VersionCount:  int(r.VersionCount),
			Downloads:     int(r.Downloads),
			SourceType:    r.SourceType,
			Published:     r.Published,
			PlatformCount: int(r.PlatformCount),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"page":      page,
		"limit":     limit,
		"total":     total,
	})
}

// GetProviderVersionsDetail returns all versions of a specific provider with platform details.
func (h *MirrorHandler) GetProviderVersionsDetail(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	var providers []models.Provider
	if err := h.db.Preload("Platforms").
		Where("namespace = ? AND name = ?", namespace, name).
		Order("version DESC").
		Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(providers) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"namespace": namespace,
		"name":      name,
		"versions":  providers,
		"total":     len(providers),
	})
}

// DeleteProvider deletes a provider and its files.
func (h *MirrorHandler) DeleteProvider(c *gin.Context) {
	id := c.Param("id")

	var provider models.Provider
	if err := h.db.First(&provider, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Delete associated platforms and files
	var platforms []models.ProviderPlatform
	h.db.Where("provider_id = ?", provider.ID).Find(&platforms)

	for _, p := range platforms {
		if p.FilePath != "" {
			_ = os.Remove(p.FilePath) // #nosec G104 - best effort cleanup
		}
	}

	h.db.Where("provider_id = ?", provider.ID).Delete(&models.ProviderPlatform{})
	h.db.Delete(&provider)

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted successfully"})
}

// ExportProvider exports a provider as a downloadable package.
// The package includes all platform binaries and a manifest file.
func (h *MirrorHandler) ExportProvider(c *gin.Context) {
	id := c.Param("id")

	var provider models.Provider
	if err := h.db.Preload("Platforms").First(&provider, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if len(provider.Platforms) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No platform binaries available for export"})
		return
	}

	// Create a temporary zip file
	zipFileName := fmt.Sprintf("terraform-provider-%s_%s_%s.zip",
		provider.Name, provider.Version, provider.Namespace)
	tempFile, err := os.CreateTemp("", "provider-export-*.zip")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp file"})
		return
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()
	defer func() { _ = tempFile.Close() }()

	zipWriter := zip.NewWriter(tempFile)

	// Create manifest
	manifest := ProviderExportManifest{
		Namespace:   provider.Namespace,
		Name:        provider.Name,
		Version:     provider.Version,
		Description: provider.Description,
		SourceType:  string(provider.SourceType),
		Protocols:   provider.Protocols,
		ExportedAt:  time.Now(),
		Platforms:   make([]PlatformManifest, 0),
	}

	// Add each platform binary to the zip
	for _, platform := range provider.Platforms {
		if platform.FilePath == "" {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(platform.FilePath); os.IsNotExist(err) {
			continue
		}

		// Open the source file
		srcFile, err := os.Open(platform.FilePath) // #nosec G304 - path is from database record
		if err != nil {
			continue
		}

		// Create entry in zip with relative path
		entryName := fmt.Sprintf("%s/%s/%s", platform.OS, platform.Arch, platform.Filename)
		writer, err := zipWriter.Create(entryName)
		if err != nil {
			_ = srcFile.Close() // #nosec G104 - best effort cleanup
			continue
		}

		// Copy file content
		_, err = io.Copy(writer, srcFile)
		_ = srcFile.Close() // #nosec G104 - best effort cleanup
		if err != nil {
			continue
		}

		// Add to manifest
		manifest.Platforms = append(manifest.Platforms, PlatformManifest{
			OS:        platform.OS,
			Arch:      platform.Arch,
			Filename:  platform.Filename,
			SHA256Sum: platform.SHA256Sum,
			FileSize:  platform.FileSize,
			ZipPath:   entryName,
		})
	}

	if len(manifest.Platforms) == 0 {
		_ = zipWriter.Close() // #nosec G104 - best effort cleanup
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid platform binaries found"})
		return
	}

	// Add manifest.json to zip
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	manifestWriter, err := zipWriter.Create("manifest.json")
	if err == nil {
		_, _ = manifestWriter.Write(manifestJSON) // #nosec G104 - zip write errors caught on Close
	}

	// Close zip writer
	if err := zipWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create zip file"})
		return
	}

	// Get file size
	_, _ = tempFile.Seek(0, 0) // #nosec G104 - seek on temp file
	fileInfo, _ := tempFile.Stat()

	// Set headers for download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", zipFileName))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	c.Header("Content-Transfer-Encoding", "binary")

	// Stream the file
	c.File(tempFile.Name())
}

// ProviderExportManifest contains metadata for exported provider package.
type ProviderExportManifest struct {
	Namespace   string             `json:"namespace"`
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	SourceType  string             `json:"source_type"`
	Protocols   string             `json:"protocols"`
	ExportedAt  time.Time          `json:"exported_at"`
	Platforms   []PlatformManifest `json:"platforms"`
}

// PlatformManifest contains platform-specific metadata.
type PlatformManifest struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Filename  string `json:"filename"`
	SHA256Sum string `json:"sha256sum"`
	FileSize  int64  `json:"file_size"`
	ZipPath   string `json:"zip_path"`
}

// ImportProvider imports a provider from an exported package.
func (h *MirrorHandler) ImportProvider(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() { _ = file.Close() }()

	// Save and open zip file
	zipReader, cleanup, err := h.saveAndOpenZip(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer cleanup()

	// Read manifest from zip
	manifest, err := h.readManifestFromZip(zipReader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create or update provider
	provider, err := h.getOrCreateImportedProvider(manifest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract and save platforms
	importedPlatforms := h.extractPlatformsFromZip(zipReader, manifest, provider.ID)

	if len(importedPlatforms) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No platforms were imported"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Provider imported successfully",
		"provider":  provider,
		"platforms": importedPlatforms,
		"file_name": header.Filename,
	})
}

// saveAndOpenZip saves the uploaded file and opens it as a zip reader.
func (h *MirrorHandler) saveAndOpenZip(file io.Reader) (*zip.ReadCloser, func(), error) {
	tempFile, err := os.CreateTemp("", "provider-import-*.zip")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file")
	}

	if _, err := io.Copy(tempFile, file); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return nil, nil, fmt.Errorf("failed to save uploaded file")
	}
	_ = tempFile.Close()

	zipReader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		_ = os.Remove(tempFile.Name())
		return nil, nil, fmt.Errorf("invalid zip file")
	}

	cleanup := func() {
		_ = zipReader.Close() // #nosec G104 - best effort cleanup
		_ = os.Remove(tempFile.Name())
	}
	return zipReader, cleanup, nil
}

// readManifestFromZip reads the manifest.json from a zip file.
func (h *MirrorHandler) readManifestFromZip(zipReader *zip.ReadCloser) (*ProviderExportManifest, error) {
	for _, f := range zipReader.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			var manifest ProviderExportManifest
			err = json.NewDecoder(rc).Decode(&manifest)
			_ = rc.Close()
			if err == nil {
				return &manifest, nil
			}
		}
	}
	return nil, fmt.Errorf("manifest.json not found in package")
}

// getOrCreateImportedProvider creates or retrieves a provider for import.
func (h *MirrorHandler) getOrCreateImportedProvider(manifest *ProviderExportManifest) (*models.Provider, error) {
	var provider models.Provider
	result := h.db.Where("namespace = ? AND name = ? AND version = ?",
		manifest.Namespace, manifest.Name, manifest.Version).First(&provider)

	if result.Error == gorm.ErrRecordNotFound {
		provider = models.Provider{
			Namespace:   manifest.Namespace,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			SourceType:  models.SourceType(manifest.SourceType),
			Published:   time.Now(),
			Protocols:   manifest.Protocols,
		}
		if err := h.db.Create(&provider).Error; err != nil {
			return nil, err
		}
	}
	return &provider, nil
}

// extractPlatformsFromZip extracts platform binaries from the zip file.
func (h *MirrorHandler) extractPlatformsFromZip(zipReader *zip.ReadCloser, manifest *ProviderExportManifest, providerID uint) []PlatformManifest {
	const maxFileSize = 500 * 1024 * 1024 // 500MB max per file
	importedPlatforms := make([]PlatformManifest, 0)

	for _, pm := range manifest.Platforms {
		zipFile := h.findFileInZip(zipReader, pm.ZipPath)
		if zipFile == nil || zipFile.UncompressedSize64 > maxFileSize {
			continue
		}

		filePath, err := h.extractZipFile(zipFile, manifest.Namespace, manifest.Name, manifest.Version, pm)
		if err != nil {
			continue
		}

		h.saveImportedPlatform(providerID, pm, filePath)
		importedPlatforms = append(importedPlatforms, pm)
	}
	return importedPlatforms
}

// findFileInZip finds a file in the zip by path.
func (h *MirrorHandler) findFileInZip(zipReader *zip.ReadCloser, path string) *zip.File {
	for _, f := range zipReader.File {
		if f.Name == path {
			return f
		}
	}
	return nil
}

// extractZipFile extracts a single file from the zip.
func (h *MirrorHandler) extractZipFile(zipFile *zip.File, namespace, name, version string, pm PlatformManifest) (string, error) {
	const maxFileSize = 500 * 1024 * 1024

	// Build safe directory path using validated components
	dirPath, err := proxy.BuildSafeProviderPath(h.storagePath, namespace, name, version, pm.OS, pm.Arch)
	if err != nil {
		return "", fmt.Errorf("invalid path components: %w", err)
	}

	if err := os.MkdirAll(dirPath, 0750); err != nil {
		return "", err
	}

	// Sanitize filename
	safeFilename, err := proxy.SanitizeFilename(pm.Filename)
	if err != nil {
		return "", fmt.Errorf("invalid filename: %w", err)
	}

	filePath := filepath.Join(dirPath, safeFilename)
	// #nosec G304 -- filePath is constructed from validated components via BuildSafeProviderPath and SanitizeFilename
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", err
	}

	rc, err := zipFile.Open()
	if err != nil {
		_ = outFile.Close()
		return "", err
	}

	_, err = io.Copy(outFile, io.LimitReader(rc, int64(maxFileSize)))
	_ = rc.Close()
	_ = outFile.Close()

	if err != nil {
		_ = os.Remove(filePath)
		return "", err
	}
	return filePath, nil
}

// saveImportedPlatform saves an imported platform to the database.
func (h *MirrorHandler) saveImportedPlatform(providerID uint, pm PlatformManifest, filePath string) {
	fileSize := getFileSize(filePath)

	var existingPlatform models.ProviderPlatform
	if err := h.db.Where("provider_id = ? AND os = ? AND arch = ?",
		providerID, pm.OS, pm.Arch).First(&existingPlatform).Error; err == nil {
		existingPlatform.Filename = pm.Filename
		existingPlatform.FilePath = filePath
		existingPlatform.SHA256Sum = pm.SHA256Sum
		existingPlatform.FileSize = fileSize
		h.db.Save(&existingPlatform)
	} else {
		platform := models.ProviderPlatform{
			ProviderID: providerID,
			OS:         pm.OS,
			Arch:       pm.Arch,
			Filename:   pm.Filename,
			FilePath:   filePath,
			SHA256Sum:  pm.SHA256Sum,
			FileSize:   fileSize,
		}
		h.db.Create(&platform)
	}
}
