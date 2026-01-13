// Package api provides HTTP handlers for the API.
package api

import (
	"net/http"
	"strconv"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler contains dependencies for HTTP handlers.
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new Handler instance.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ProviderSummary represents a provider with aggregated version info.
type ProviderSummary struct {
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	LatestVersion string `json:"version"`
	VersionCount  int    `json:"version_count"`
	Downloads     int    `json:"downloads"`
	Published     string `json:"published"`
}

// ListProviders returns a list of unique providers (grouped by namespace/name).
func (h *Handler) ListProviders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Query to get unique providers grouped by namespace/name
	var results []struct {
		Namespace     string
		Name          string
		Description   string
		LatestVersion string
		VersionCount  int64
		Downloads     int64
		Published     string
	}

	baseQuery := h.db.Model(&models.Provider{})

	if namespace := c.Query("namespace"); namespace != "" {
		baseQuery = baseQuery.Where("namespace = ?", namespace)
	}

	if name := c.Query("name"); name != "" {
		baseQuery = baseQuery.Where("name LIKE ?", "%"+name+"%")
	}

	// Count unique providers
	var total int64
	h.db.Model(&models.Provider{}).Select("COUNT(DISTINCT namespace || '/' || name)").Scan(&total)

	// Get aggregated provider info
	if err := baseQuery.Select(`
		namespace,
		name,
		MAX(description) as description,
		MAX(version) as latest_version,
		COUNT(DISTINCT version) as version_count,
		SUM(downloads) as downloads,
		MAX(published) as published
	`).
		Group("namespace, name").
		Order("MAX(created_at) DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to ProviderSummary
	providers := make([]ProviderSummary, len(results))
	for i, r := range results {
		providers[i] = ProviderSummary{
			Namespace:     r.Namespace,
			Name:          r.Name,
			Description:   r.Description,
			LatestVersion: r.LatestVersion,
			VersionCount:  int(r.VersionCount),
			Downloads:     int(r.Downloads),
			Published:     r.Published,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"page":      page,
		"limit":     limit,
		"total":     total,
	})
}

// GetProvider returns a specific provider.
func (h *Handler) GetProvider(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Param("version")

	var provider models.Provider
	if err := h.db.Where("namespace = ? AND name = ? AND version = ?",
		namespace, name, version).First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// CreateProvider creates a new provider.
func (h *Handler) CreateProvider(c *gin.Context) {
	var provider models.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Create(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// ListModules returns a list of modules.
func (h *Handler) ListModules(c *gin.Context) {
	var modules []models.Module

	query := h.db.Model(&models.Module{})

	if namespace := c.Query("namespace"); namespace != "" {
		query = query.Where("namespace = ?", namespace)
	}

	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&modules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"modules": modules,
		"page":    page,
		"limit":   limit,
		"total":   total,
	})
}

// GetModule returns a specific module.
func (h *Handler) GetModule(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")
	version := c.Param("version")

	var module models.Module
	if err := h.db.Where("namespace = ? AND name = ? AND provider = ? AND version = ?",
		namespace, name, provider, version).First(&module).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, module)
}

// SearchProviders searches for providers.
func (h *Handler) SearchProviders(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	var providers []models.Provider
	if err := h.db.Where("name LIKE ? OR description LIKE ?",
		"%"+q+"%", "%"+q+"%").Limit(20).Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// HealthCheck returns the health status of the service.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "vc-terraform-registry",
	})
}
