// Package config provides configuration management for the application.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// A real signing key (>= minSecretKeyLen, not a known placeholder) must be
	// present for Load to succeed in the default (release) mode; supplying it
	// via the environment also exercises the env-override path that was broken.
	const testKey = "unit-test-signing-key-0123456789ABCDEF"
	t.Setenv("AUTH_SECRETKEY", testKey)

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
		if cfg.Auth.SecretKey != testKey {
			t.Errorf("Auth.SecretKey = %q, want %q", cfg.Auth.SecretKey, testKey)
		}
	})

	t.Run("log defaults", func(t *testing.T) {
		if cfg.Log.Level != "info" {
			t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
		}
	})
}

// TestLoad_EnvOverride is the regression test for the AutomaticEnv bug: nested
// keys such as auth.secretkey could never be set from AUTH_SECRETKEY, so every
// environment override was silently dropped and the insecure defaults won.
func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("SERVER_PORT", "19090")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_MODE", "debug")
	t.Setenv("DATABASE_URL", "sqlite:/tmp/override.db")
	t.Setenv("STORAGE_PATH", "/tmp/override-storage")
	t.Setenv("AUTH_SECRETKEY", "env-supplied-secret")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	checks := map[string][2]string{
		"Server.Port":    {cfg.Server.Port, "19090"},
		"Server.Host":    {cfg.Server.Host, "127.0.0.1"},
		"Server.Mode":    {cfg.Server.Mode, "debug"},
		"Database.URL":   {cfg.Database.URL, "sqlite:/tmp/override.db"},
		"Storage.Path":   {cfg.Storage.Path, "/tmp/override-storage"},
		"Auth.SecretKey": {cfg.Auth.SecretKey, "env-supplied-secret"},
		"Log.Level":      {cfg.Log.Level, "debug"},
	}
	for field, got := range checks {
		if got[0] != got[1] {
			t.Errorf("%s = %q from env, want %q", field, got[0], got[1])
		}
	}
}

// TestLoad_RejectsInsecureSecretInRelease locks in the fail-closed guard: a
// release-mode deployment must refuse to start on any published placeholder key
// or a key shorter than the minimum, since those are the values that actually
// ship in the example manifests.
func TestLoad_RejectsInsecureSecretInRelease(t *testing.T) {
	cases := map[string]string{
		"viper default":      "change-me-in-production",
		"docker-compose key": "dev-secret-key-change-in-production",
		"docker-compose dev": "dev-secret-key",
		"k8s secret key":     "change-me-in-production-use-secure-random-key",
		"k8s readme key":     "your-secure-key-here",
		"too short":          "short-key",
		"empty":              "",
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("SERVER_MODE", "release")
			t.Setenv("AUTH_ENABLED", "true")
			t.Setenv("AUTH_SECRETKEY", key)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() succeeded with insecure key %q in release mode; want error", key)
			}
		})
	}
}

// TestLoad_AcceptsStrongSecretInRelease is the positive control: a sufficiently
// long, non-placeholder key must be accepted in release mode.
func TestLoad_AcceptsStrongSecretInRelease(t *testing.T) {
	t.Setenv("SERVER_MODE", "release")
	t.Setenv("AUTH_ENABLED", "true")
	t.Setenv("AUTH_SECRETKEY", "a-sufficiently-long-random-signing-key-1234")
	if _, err := Load(); err != nil {
		t.Fatalf("Load() rejected a strong key in release mode: %v", err)
	}
}

// TestLoad_AllowsDefaultSecretInDebug keeps local development frictionless: the
// same default key is a warning, not an error, when not in release mode.
func TestLoad_AllowsDefaultSecretInDebug(t *testing.T) {
	t.Setenv("SERVER_MODE", "debug")
	t.Setenv("AUTH_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.SecretKey != defaultInsecureSecretKey {
		t.Errorf("Auth.SecretKey = %q, want the default %q", cfg.Auth.SecretKey, defaultInsecureSecretKey)
	}
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
