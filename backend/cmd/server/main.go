// Package main is the entry point for the VC Terraform Registry server.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/api"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/auth"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/scheduler"
	"github.com/Veritas-Calculus/vc-terraform-registry/pkg/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
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

	router := api.SetupRouter(db, jwtManager, cfg.Auth.Enabled, storagePath)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
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
			adminPassword = "admin123"
		}
		hashedPassword, err := auth.HashPassword(adminPassword)
		if err != nil {
			log.Printf("Warning: failed to hash admin password: %v", err)
		} else {
			adminUser := models.User{
				Username: "admin",
				Email:    "admin@localhost",
				Password: hashedPassword,
				Role:     "admin",
			}
			if err := db.Create(&adminUser).Error; err != nil {
				log.Printf("Warning: failed to create admin user: %v", err)
			} else {
				log.Printf("Default admin user created (username: admin)")
			}
		}
	}

	log.Printf("Database initialized: %s", dbPath)
	return db, nil
}
