// Package config provides configuration management for the application.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any existing config
	_ = os.Unsetenv("SERVER_PORT")
	_ = os.Unsetenv("SERVER_HOST")
	_ = os.Unsetenv("DATABASE_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	t.Run("server defaults", func(t *testing.T) {
		if cfg.Server.Port != "8080" {
			t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "8080")
		}
		if cfg.Server.Host != "0.0.0.0" {
			t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
		}
		if cfg.Server.Mode != "release" {
			t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "release")
		}
	})

	t.Run("storage defaults", func(t *testing.T) {
		if cfg.Storage.Path != "/data/registry" {
			t.Errorf("Storage.Path = %q, want %q", cfg.Storage.Path, "/data/registry")
		}
		if cfg.Storage.Type != "local" {
			t.Errorf("Storage.Type = %q, want %q", cfg.Storage.Type, "local")
		}
	})

	t.Run("auth defaults", func(t *testing.T) {
		if !cfg.Auth.Enabled {
			t.Error("Auth.Enabled = false, want true")
		}
		if cfg.Auth.SecretKey != "change-me-in-production" {
			t.Errorf("Auth.SecretKey = %q, want %q", cfg.Auth.SecretKey, "change-me-in-production")
		}
	})

	t.Run("log defaults", func(t *testing.T) {
		if cfg.Log.Level != "info" {
			t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
		}
	})
}

func TestLoad_FromConfigFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configContent := `
server:
  port: "9090"
  host: "127.0.0.1"
  mode: "debug"
database:
  url: "postgres://localhost/test"
storage:
  path: "/custom/storage"
  type: "local"
auth:
  enabled: false
  secretkey: "my-secret"
log:
  level: "debug"
`
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9090")
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "debug")
	}
	if cfg.Database.URL != "postgres://localhost/test" {
		t.Errorf("Database.URL = %q, want %q", cfg.Database.URL, "postgres://localhost/test")
	}
	if cfg.Storage.Path != "/custom/storage" {
		t.Errorf("Storage.Path = %q, want %q", cfg.Storage.Path, "/custom/storage")
	}
	if cfg.Auth.Enabled {
		t.Error("Auth.Enabled = true, want false")
	}
	if cfg.Auth.SecretKey != "my-secret" {
		t.Errorf("Auth.SecretKey = %q, want %q", cfg.Auth.SecretKey, "my-secret")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
}

func TestConfig_Structs(t *testing.T) {
	t.Run("Config struct fields", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: "8080",
				Host: "localhost",
				Mode: "release",
			},
			Database: DatabaseConfig{
				URL: "sqlite:///test.db",
			},
			Storage: StorageConfig{
				Path: "/data",
				Type: "local",
			},
			Auth: AuthConfig{
				Enabled:   true,
				SecretKey: "secret",
			},
			Log: LogConfig{
				Level: "info",
			},
		}

		if cfg.Server.Port != "8080" {
			t.Error("Config struct not properly initialized")
		}
	})

	t.Run("ServerConfig struct", func(t *testing.T) {
		sc := ServerConfig{
			Port: "3000",
			Host: "0.0.0.0",
			Mode: "debug",
		}
		if sc.Port == "" || sc.Host == "" || sc.Mode == "" {
			t.Error("ServerConfig fields should not be empty")
		}
	})

	t.Run("DatabaseConfig struct", func(t *testing.T) {
		dc := DatabaseConfig{
			URL: "postgres://user:pass@localhost/db",
		}
		if dc.URL == "" {
			t.Error("DatabaseConfig.URL should not be empty")
		}
	})

	t.Run("StorageConfig struct", func(t *testing.T) {
		sc := StorageConfig{
			Path: "/var/data",
			Type: "s3",
		}
		if sc.Path == "" || sc.Type == "" {
			t.Error("StorageConfig fields should not be empty")
		}
	})

	t.Run("AuthConfig struct", func(t *testing.T) {
		ac := AuthConfig{
			Enabled:   true,
			SecretKey: "mysecret",
		}
		if !ac.Enabled {
			t.Error("AuthConfig.Enabled should be true")
		}
		if ac.SecretKey == "" {
			t.Error("AuthConfig.SecretKey should not be empty")
		}
	})

	t.Run("LogConfig struct", func(t *testing.T) {
		lc := LogConfig{
			Level: "warn",
		}
		if lc.Level == "" {
			t.Error("LogConfig.Level should not be empty")
		}
	})
}
