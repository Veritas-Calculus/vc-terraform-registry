// Package api provides HTTP routing and handlers.
package api

import (
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter configures and returns the HTTP router.
func SetupRouter(db *gorm.DB, jwtManager *auth.JWTManager, authEnabled bool, storagePath string) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	handler := NewHandler(db)
	mirrorHandler := NewMirrorHandler(db, storagePath)
	authHandler := NewAuthHandler(db, jwtManager)
	settingsHandler := NewSettingsHandler(db)
	syncHandler := NewSyncHandler(db, storagePath)
	searchHandler := NewSearchHandler(db, storagePath)

	// Terraform Registry Protocol Discovery
	router.GET("/.well-known/terraform.json", func(c *gin.Context) {
		host := c.Request.Host
		c.JSON(200, gin.H{
			"providers.v1": "/v1/providers/",
			"modules.v1":   "/v1/modules/",
			"metadata.v1":  "https://" + host + "/",
		})
	})

	// Terraform Provider Mirror Protocol
	// https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol
	mirrorProtocolHandler := NewProviderMirrorHandler(db, storagePath)
	router.GET("/registry.terraform.io/:namespace/:name/index.json", mirrorProtocolHandler.ListAvailableVersions)
	router.GET("/registry.terraform.io/:namespace/:name/:version", mirrorProtocolHandler.GetVersionArchives)

	// Terraform Provider Registry Protocol v1
	router.GET("/v1/providers/:namespace/:name/versions", mirrorHandler.GetProviderVersions)
	router.GET("/v1/providers/:namespace/:name/:version/download/:os/:arch", mirrorHandler.GetProviderDownloadInfo)
	router.GET("/v1/providers/:namespace/:name/:version/download/:os/:arch/binary", mirrorHandler.DownloadProvider)

	// Auth routes (always public)
	router.POST("/api/v1/auth/login", authHandler.Login)
	router.GET("/api/v1/auth/status", func(c *gin.Context) {
		c.JSON(200, gin.H{"auth_enabled": authEnabled})
	})

	// Public read-only routes (no auth required)
	router.GET("/health", handler.HealthCheck)
	router.GET("/api/v1/providers", handler.ListProviders)
	router.GET("/api/v1/providers/:namespace/:name/:version", handler.GetProvider)
	router.GET("/api/v1/providers/search", searchHandler.SearchProviders)
	router.GET("/api/v1/modules", handler.ListModules)
	router.GET("/api/v1/modules/:namespace/:name/:provider/:version", handler.GetModule)
	router.GET("/api/v1/mirror/providers", mirrorHandler.ListMirroredProviders)
	router.GET("/api/v1/mirror/providers/:namespace/:name", mirrorHandler.GetProviderVersionsDetail)
	router.GET("/api/v1/settings", settingsHandler.GetSettings)
	router.GET("/api/v1/sync/schedules", syncHandler.ListSchedules)

	// Protected routes (auth required for write operations)
	authorized := router.Group("/api/v1")
	authorized.Use(auth.AuthMiddleware(jwtManager))
	{
		// Auth
		authorized.GET("/auth/me", authHandler.GetCurrentUser)

		// Provider management (requires auth)
		authorized.POST("/providers", handler.CreateProvider)
		authorized.POST("/providers/upload", mirrorHandler.UploadProvider)
		authorized.DELETE("/providers/:id", mirrorHandler.DeleteProvider)

		// Mirror operations (requires auth)
		authorized.GET("/mirror/upstream/:namespace/:name", mirrorHandler.ListUpstreamVersions)
		authorized.POST("/mirror/:namespace/:name", mirrorHandler.MirrorProvider)
		authorized.GET("/mirror/:namespace/:name/stream", mirrorHandler.MirrorProviderWithProgress)
		authorized.GET("/mirror/export/:id", mirrorHandler.ExportProvider)
		authorized.POST("/mirror/import", mirrorHandler.ImportProvider)

		// Settings (requires auth)
		authorized.PUT("/settings", settingsHandler.UpdateSettings)

		// Sync schedules (requires auth)
		authorized.POST("/sync/schedules", syncHandler.CreateSchedule)
		authorized.PUT("/sync/schedules/:id", syncHandler.UpdateSchedule)
		authorized.DELETE("/sync/schedules/:id", syncHandler.DeleteSchedule)
		authorized.POST("/sync/schedules/:id/run", syncHandler.RunScheduleNow)

		// Module management
		authorized.POST("/modules", handler.CreateProvider)
	}

	return router
}
