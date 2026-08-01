// Package main is the entry point for the VC Terraform Registry server.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/api"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/auth"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/scheduler"
	"github.com/Veritas-Calculus/vc-terraform-registry/pkg/config"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Apply the configured Gin mode. Left unset, Gin defaults to debug mode,
	// which dumps the full route table and request logs and warns on startup.
	switch cfg.Server.Mode {
	case "debug", "test":
		gin.SetMode(cfg.Server.Mode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	jwtManager := auth.NewJWTManager(cfg.Auth.SecretKey, 24*time.Hour)

	storagePath := cfg.Storage.Path
	if storagePath == "" {
		storagePath = "./data/providers"
	}
	if err := os.MkdirAll(storagePath, 0750); err != nil { // #nosec G301 - storage directory needs group access
		log.Fatalf("Failed to create storage directory: %v", err)
	}
	storagePath, _ = filepath.Abs(storagePath)
	log.Printf("Storage path: %s", storagePath)

	syncScheduler := scheduler.New(db, storagePath)
	if err := syncScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start scheduler: %v", err)
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		syncScheduler.Stop()
		os.Exit(0)
	}()

	corsOrigins := parseCORSOrigins(cfg.CORS.AllowedOrigins)
	router := api.SetupRouter(db, jwtManager, cfg.Auth.Enabled, storagePath, corsOrigins)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// generatePassword returns a URL-safe random password with at least nBytes of
// entropy, suitable for the auto-provisioned admin account.
func generatePassword(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// parseCORSOrigins splits the comma-separated CORS origins config into a
// trimmed, non-empty slice. An empty or "*" value yields ["*"] (allow any).
func parseCORSOrigins(raw string) []string {
	var origins []string
	for part := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	dbPath := cfg.Database.URL
	if dbPath == "" {
		dbPath = "/data/registry.db"
	}

	if len(dbPath) > 7 && dbPath[:7] == "sqlite:" {
		dbPath = dbPath[7:]
	}

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0750); err != nil { // #nosec G301 - database directory needs group access
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&models.Provider{},
		&models.Module{},
		&models.User{},
		&models.ProviderPlatform{},
		&models.MirrorConfig{},
		&models.Settings{},
		&models.SyncSchedule{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create default admin user if no users exist
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			if cfg.Auth.Enabled {
				// Fail closed rather than seed a well-known password or log a
				// generated one (logging a secret is itself a leak). The operator
				// must choose the initial admin password explicitly.
				return nil, fmt.Errorf(
					"ADMIN_PASSWORD is not set; set it to create the initial admin account " +
						"(e.g. `openssl rand -base64 24`)")
			}
			// Authentication is disabled (development): the admin credential gates
			// nothing, so seed a random password that is never logged.
			adminPassword, err = generatePassword(24)
			if err != nil {
				return nil, fmt.Errorf("failed to generate admin password: %w", err)
			}
		}
		hashedPassword, err := auth.HashPassword(adminPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to hash admin password: %w", err)
		}
		adminUser := models.User{
			Username: "admin",
			Email:    "admin@localhost",
			Password: hashedPassword,
			Role:     "admin",
		}
		if err := db.Create(&adminUser).Error; err != nil {
			return nil, fmt.Errorf("failed to create admin user: %w", err)
		}
		log.Printf("Initial admin user 'admin' created")
	}

	log.Printf("Database initialized: %s", dbPath)
	return db, nil
}
