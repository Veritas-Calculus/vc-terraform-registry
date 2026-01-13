package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddleware(t *testing.T) {
	// Create a test JWTManager
	jwtManager := NewJWTManager("test-secret-key", time.Hour)

	t.Run("missing authorization header", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat token123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid token in header", func(t *testing.T) {
		// Generate a valid token first
		token, err := jwtManager.Generate(1, "testuser", "admin")
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))
		router.GET("/protected", func(c *gin.Context) {
			username, _ := c.Get("username")
			role, _ := c.Get("role")
			c.JSON(http.StatusOK, gin.H{
				"username": username,
				"role":     role,
			})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("valid token in query parameter", func(t *testing.T) {
		token, err := jwtManager.Generate(1, "testuser", "user")
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))
		router.GET("/protected", func(c *gin.Context) {
			username, _ := c.Get("username")
			c.JSON(http.StatusOK, gin.H{"username": username})
		})

		req := httptest.NewRequest("GET", "/protected?token="+token, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestRequireRole(t *testing.T) {
	jwtManager := NewJWTManager("test-secret-key", time.Hour)

	t.Run("user has required role", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))
		router.Use(RequireRole("admin"))
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		token, _ := jwtManager.Generate(1, "admin", "admin")
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("user lacks required role", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))
		router.Use(RequireRole("admin"))
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		token, _ := jwtManager.Generate(2, "user", "user")
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
		}
	})

	t.Run("no role in context", func(t *testing.T) {
		router := gin.New()
		// Directly use RequireRole without AuthMiddleware
		router.Use(RequireRole("admin"))
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
		}
	})
}

func TestMultipleRoles(t *testing.T) {
	jwtManager := NewJWTManager("test-secret-key", time.Hour)

	t.Run("admin can access admin endpoint", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtManager))

		// Create a route group that allows only admin role
		adminGroup := router.Group("/")
		adminGroup.Use(RequireRole("admin"))
		adminGroup.GET("/admin-only", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "admin access"})
		})

		token, _ := jwtManager.Generate(1, "superuser", "admin")
		req := httptest.NewRequest("GET", "/admin-only", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})
}
