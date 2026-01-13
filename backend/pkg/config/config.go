// Package config provides configuration management for the application.
package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Auth     AuthConfig
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

// LogConfig contains logging configuration.
type LogConfig struct {
	Level string
}

// Load reads configuration from environment variables and config files.
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("database.url", "sqlite:///data/registry.db")
	viper.SetDefault("storage.path", "/data/registry")
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("auth.enabled", true)
	viper.SetDefault("auth.secretkey", "change-me-in-production")
	viper.SetDefault("log.level", "info")

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

	return &cfg, nil
}
