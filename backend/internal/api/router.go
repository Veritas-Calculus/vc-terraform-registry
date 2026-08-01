// Package api provides HTTP routing and handlers.
package api

import (
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter configures and returns the HTTP router.
func SetupRouter(db *gorm.DB, jwtManager *auth.JWTManager, authEnabled bool, storagePath string, corsAllowedOrigins []string) *gin.Engine {
	// Build the engine explicitly so the access log never records the JWTs that
	// SSE and export endpoints pass in the query string.
	router := gin.New()
	router.Use(redactingLogger(), gin.Recovery())

	router.Use(corsMiddleware(corsAllowedOrigins))

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

	// Auth routes (always public). Login is rate limited per client IP to stop
	// unbounded online password guessing and bcrypt CPU exhaustion.
	loginLimiter := newRateLimiter(10, time.Minute)
	router.POST("/api/v1/auth/login", rateLimitMiddleware(loginLimiter, time.Now), authHandler.Login)
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

	// Authenticated routes: any valid token. /auth/me only exposes the caller's
	// own identity, so it does not need an elevated role.
	authenticated := router.Group("/api/v1")
	authenticated.Use(auth.AuthMiddleware(jwtManager))
	{
		authenticated.GET("/auth/me", authHandler.GetCurrentUser)
	}

	// Admin routes: valid token AND the admin role. Every state-changing
	// management operation lives here. Without the RequireRole gate the role
	// claim was decorative and any authenticated principal was fully privileged.
	admin := router.Group("/api/v1")
	admin.Use(auth.AuthMiddleware(jwtManager), auth.RequireRole("admin"))
	{
		// Provider management
		admin.POST("/providers", handler.CreateProvider)
		admin.POST("/providers/upload", mirrorHandler.UploadProvider)
		admin.DELETE("/providers/:id", mirrorHandler.DeleteProvider)

		// Mirror operations
		admin.GET("/mirror/upstream/:namespace/:name", mirrorHandler.ListUpstreamVersions)
		admin.POST("/mirror/:namespace/:name", mirrorHandler.MirrorProvider)
		admin.GET("/mirror/:namespace/:name/stream", mirrorHandler.MirrorProviderWithProgress)
		admin.GET("/mirror/export/:id", mirrorHandler.ExportProvider)
		admin.POST("/mirror/import", mirrorHandler.ImportProvider)

		// Settings
		admin.PUT("/settings", settingsHandler.UpdateSettings)

		// Sync schedules
		admin.POST("/sync/schedules", syncHandler.CreateSchedule)
		admin.PUT("/sync/schedules/:id", syncHandler.UpdateSchedule)
		admin.DELETE("/sync/schedules/:id", syncHandler.DeleteSchedule)
		admin.POST("/sync/schedules/:id/run", syncHandler.RunScheduleNow)

		// Module management
		admin.POST("/modules", handler.CreateProvider)
	}

	return router
}

// corsMiddleware returns a CORS handler.
//
// The API authenticates with a Bearer token in the Authorization header, not
// cookies, so it never needs Access-Control-Allow-Credentials. Emitting that
// header alongside a wildcard origin is both a spec violation (browsers reject
// it) and a latent misconfiguration, so this handler never sets it.
//
//   - allowedOrigins == ["*"] (default): any origin is allowed, no credentials.
//   - explicit list: the request Origin is echoed only when it matches, and
//     Vary: Origin is set so caches key on the origin.
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowAny := len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*")
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if allowAny {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
			c.Writer.Header().Add("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
