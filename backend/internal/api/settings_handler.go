// Package api provides HTTP handlers for settings management.
package api

import (
	"net/http"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SettingsHandler handles settings-related HTTP requests.
type SettingsHandler struct {
	db *gorm.DB
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

// SettingsResponse represents the settings API response.
type SettingsResponse struct {
	AllowOnlineSearch  bool   `json:"allow_online_search"`
	DefaultUpstreamURL string `json:"default_upstream_url"`
	RegistryURL        string `json:"registry_url"`
	ProxyEnabled       bool   `json:"proxy_enabled"`
	ProxyURL           string `json:"proxy_url"`
	ProxyType          string `json:"proxy_type"`
}

// UpdateSettingsRequest represents the request to update settings.
type UpdateSettingsRequest struct {
	AllowOnlineSearch  *bool   `json:"allow_online_search"`
	DefaultUpstreamURL *string `json:"default_upstream_url"`
	RegistryURL        *string `json:"registry_url"`
	ProxyEnabled       *bool   `json:"proxy_enabled"`
	ProxyURL           *string `json:"proxy_url"`
	ProxyType          *string `json:"proxy_type"`
}

// GetSettings returns the current application settings.
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	var settings models.Settings
	result := h.db.First(&settings)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			settings = models.Settings{
				AllowOnlineSearch:  true,
				DefaultUpstreamURL: "https://registry.terraform.io",
			}
			h.db.Create(&settings)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
			return
		}
	}

	c.JSON(http.StatusOK, SettingsResponse{
		AllowOnlineSearch:  settings.AllowOnlineSearch,
		DefaultUpstreamURL: settings.DefaultUpstreamURL,
		RegistryURL:        settings.RegistryURL,
		ProxyEnabled:       settings.ProxyEnabled,
		ProxyURL:           settings.ProxyURL,
		ProxyType:          settings.ProxyType,
	})
}

// UpdateSettings updates the application settings.
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var settings models.Settings
	result := h.db.First(&settings)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			settings = models.Settings{
				AllowOnlineSearch:  true,
				DefaultUpstreamURL: "https://registry.terraform.io",
			}
			h.db.Create(&settings)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
			return
		}
	}

	if req.AllowOnlineSearch != nil {
		settings.AllowOnlineSearch = *req.AllowOnlineSearch
	}
	if req.DefaultUpstreamURL != nil {
		settings.DefaultUpstreamURL = *req.DefaultUpstreamURL
	}
	if req.RegistryURL != nil {
		settings.RegistryURL = *req.RegistryURL
	}
	if req.ProxyEnabled != nil {
		settings.ProxyEnabled = *req.ProxyEnabled
	}
	if req.ProxyURL != nil {
		settings.ProxyURL = *req.ProxyURL
	}
	if req.ProxyType != nil {
		settings.ProxyType = *req.ProxyType
	}

	if err := h.db.Save(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
		return
	}

	c.JSON(http.StatusOK, SettingsResponse{
		AllowOnlineSearch:  settings.AllowOnlineSearch,
		DefaultUpstreamURL: settings.DefaultUpstreamURL,
		RegistryURL:        settings.RegistryURL,
		ProxyEnabled:       settings.ProxyEnabled,
		ProxyURL:           settings.ProxyURL,
		ProxyType:          settings.ProxyType,
	})
}
