// Package api provides HTTP handlers for the provider search.
package api

import (
	"net/http"
	"strconv"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/proxy"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SearchHandler handles provider search operations.
type SearchHandler struct {
	db           *gorm.DB
	proxyService *proxy.ProxyService
}

// NewSearchHandler creates a new SearchHandler instance.
func NewSearchHandler(db *gorm.DB, storagePath string) *SearchHandler {
	h := &SearchHandler{
		db:           db,
		proxyService: proxy.NewProxyService(storagePath, ""),
	}
	h.refreshProxySettings()
	return h
}

// refreshProxySettings loads proxy settings from database and updates the proxy service.
func (h *SearchHandler) refreshProxySettings() {
	var settings models.Settings
	if err := h.db.First(&settings).Error; err == nil {
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}
}

// ProviderSearchResult represents a provider in search results.
type ProviderSearchResult struct {
	ID          uint   `json:"id,omitempty"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Downloads   int64  `json:"downloads"`
	Source      string `json:"source"` // "local" or "upstream"
	IsCached    bool   `json:"is_cached"`
	Tier        string `json:"tier"` // "official", "partner", or "community"
}

// determineTier returns the tier for a provider based on namespace and source.
func determineTier(namespace, source string) string {
	if namespace == "hashicorp" {
		return "official"
	}
	if source == "partner" {
		return "partner"
	}
	return "community"
}

// searchLocalProviders searches for providers in the local database.
func (h *SearchHandler) searchLocalProviders(query string, offset, limit int) ([]models.Provider, int64, error) {
	var providers []models.Provider
	dbQuery := h.db.Model(&models.Provider{})

	if query != "" {
		dbQuery = dbQuery.Where("name LIKE ? OR namespace LIKE ?", "%"+query+"%", "%"+query+"%")
	}

	var total int64
	dbQuery.Count(&total)

	err := dbQuery.Offset(offset).Limit(limit).Order("downloads DESC").Find(&providers).Error
	return providers, total, err
}

// buildLocalResults converts local providers to search results.
func (h *SearchHandler) buildLocalResults(providers []models.Provider) ([]ProviderSearchResult, map[string]bool) {
	results := make([]ProviderSearchResult, 0, len(providers))
	nameMap := make(map[string]bool)

	for _, p := range providers {
		key := p.Namespace + "/" + p.Name
		if !nameMap[key] {
			nameMap[key] = true
			results = append(results, ProviderSearchResult{
				ID:          p.ID,
				Namespace:   p.Namespace,
				Name:        p.Name,
				Description: p.Description,
				Downloads:   p.Downloads,
				Source:      "local",
				IsCached:    true,
				Tier:        determineTier(p.Namespace, ""),
			})
		}
	}
	return results, nameMap
}

// appendUpstreamResults adds upstream providers to results if not already present.
func (h *SearchHandler) appendUpstreamResults(results []ProviderSearchResult, nameMap map[string]bool, query string, remaining int) []ProviderSearchResult {
	upstreamResults, err := h.proxyService.SearchProviders(query, remaining)
	if err != nil || upstreamResults == nil {
		return results
	}

	for _, p := range upstreamResults.Providers {
		key := p.Namespace + "/" + p.Name
		if !nameMap[key] {
			nameMap[key] = true
			results = append(results, ProviderSearchResult{
				Namespace:   p.Namespace,
				Name:        p.Name,
				Description: p.Description,
				Downloads:   p.Downloads,
				Source:      "upstream",
				IsCached:    false,
				Tier:        determineTier(p.Namespace, p.Source),
			})
		}
	}
	return results
}

// SearchProviders searches for providers locally and optionally from upstream.
func (h *SearchHandler) SearchProviders(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		query = c.Query("name")
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Get local providers
	localProviders, localTotal, err := h.searchLocalProviders(query, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build results from local providers
	results, nameMap := h.buildLocalResults(localProviders)

	// Check if online search is allowed
	var settings models.Settings
	allowOnline := true
	if err := h.db.First(&settings).Error; err == nil {
		allowOnline = settings.AllowOnlineSearch
		h.proxyService.SetProxy(settings.ProxyEnabled, settings.ProxyURL, settings.ProxyType)
	}

	// If we have a query and online search is allowed, search upstream too
	if query != "" && allowOnline && len(results) < limit {
		results = h.appendUpstreamResults(results, nameMap, query, limit-len(results))
	}

	c.JSON(http.StatusOK, gin.H{
		"providers":     results,
		"page":          page,
		"limit":         limit,
		"total":         len(results),
		"local_total":   localTotal,
		"online_search": allowOnline,
	})
}
