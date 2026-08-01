// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// defaultInsecureSecretKey is the placeholder JWT signing key shipped as the
// viper default. It is public (it lives in this source file), so it must never
// be used to sign real tokens.
const defaultInsecureSecretKey = "change-me-in-production"

// minSecretKeyLen is the minimum acceptable length for the JWT signing key in
// release mode. HS256 keys shorter than 32 bytes are brute-forceable.
const minSecretKeyLen = 32

// knownInsecureSecretKeys are placeholder signing keys published in this repo's
// example manifests (docker-compose, k8s, docs). Every one of them is public,
// so a deployment using any of them can have admin tokens forged offline. They
// are rejected regardless of length, because some are longer than the minimum
// yet still public. Keep this list in sync with the example files.
var knownInsecureSecretKeys = map[string]bool{
	"":                                    true, // unset
	"change-me-in-production":             true, // viper default / .env.example
	"dev-secret-key-change-in-production": true, // docker-compose.yml
	"dev-secret-key":                      true, // docker-compose.dev.yml
	"change-me-in-production-use-secure-random-key": true, // k8s/secret.yaml
	"your-secure-key-here":                          true, // k8s/README.md
}

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Log      LogConfig
}

// ServerConfig contains server-related configuration.
type ServerConfig struct {
	Port string
	Host string
	Mode string
}

// DatabaseConfig contains database connection settings.
type DatabaseConfig struct {
	URL string
}

// StorageConfig contains storage backend configuration.
type StorageConfig struct {
	Path string
	Type string
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	Enabled   bool
	SecretKey string
}

// CORSConfig contains cross-origin resource sharing settings.
type CORSConfig struct {
	// AllowedOrigins is a comma-separated list of permitted origins. The
	// special value "*" allows any origin (without credentials, since the API
	// authenticates via a Bearer token rather than cookies). Set it to an
	// explicit list to restrict cross-origin browser access.
	AllowedOrigins string
}

// LogConfig contains logging configuration.
type LogConfig struct {
	Level string
}

// Load reads configuration from environment variables and config files.
func Load() (*Config, error) {
	// Viper keeps global state. Reset it so Load is self-contained and repeated
	// calls (or another package that touched viper) cannot leak a previously
	// read config file or binding into this configuration.
	viper.Reset()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Bridge nested config keys (e.g. "auth.secretkey") to flat environment
	// variables (e.g. "AUTH_SECRETKEY"). Without this replacer AutomaticEnv can
	// never match a dotted key against an underscore env var, so every value
	// supplied via the environment (AUTH_SECRETKEY, DATABASE_URL, STORAGE_PATH,
	// ...) was silently ignored and the insecure defaults were always used.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// AutomaticEnv only feeds Get/Unmarshal for keys Viper already knows about,
	// so bind every key explicitly. This makes env overrides deterministic and
	// independent of whether a default or config file happens to be present.
	for _, key := range []string{
		"server.port", "server.host", "server.mode",
		"database.url",
		"storage.path", "storage.type",
		"auth.enabled", "auth.secretkey",
		"cors.allowedorigins",
		"log.level",
	} {
		// BindEnv derives the env var name from the key via the replacer above.
		_ = viper.BindEnv(key)
	}

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("database.url", "sqlite:///data/registry.db")
	viper.SetDefault("storage.path", "/data/registry")
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("auth.enabled", true)
	viper.SetDefault("auth.secretkey", defaultInsecureSecretKey)
	viper.SetDefault("cors.allowedorigins", "*")
	viper.SetDefault("log.level", "info")

	// Give CORS a readable env var name decoupled from the dotted key.
	_ = viper.BindEnv("cors.allowedorigins", "CORS_ALLOWED_ORIGINS")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		log.Println("No config file found, using defaults and environment variables")
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Fail closed when authentication is enabled but the JWT signing key is a
	// known public placeholder or too short. Any deployment using one of these
	// keys can have admin tokens forged offline, since the keys are published in
	// this repo's example manifests. In release mode this is a hard error; in
	// dev mode it is a loud warning so local development stays frictionless.
	if cfg.Auth.Enabled {
		insecure := knownInsecureSecretKeys[cfg.Auth.SecretKey] || len(cfg.Auth.SecretKey) < minSecretKeyLen
		if insecure {
			if cfg.Server.Mode == "release" {
				return nil, fmt.Errorf(
					"AUTH_SECRETKEY is missing, a known placeholder, or shorter than %d bytes; "+
						"set a strong random key (e.g. `openssl rand -base64 32`) before starting in release mode",
					minSecretKeyLen)
			}
			log.Printf("WARNING: AUTH_SECRETKEY is insecure (a known placeholder or shorter than %d bytes). "+
				"Tokens can be forged. Set a strong random key before any non-local use.", minSecretKeyLen)
		}
	}

	return &cfg, nil
}
