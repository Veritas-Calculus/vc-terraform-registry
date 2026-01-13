package main

import (
	"testing"

	"github.com/Veritas-Calculus/vc-terraform-registry/pkg/config"
)

func TestInitDatabase(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: "sqlite:///tmp/test.db",
		},
	}

	db, err := initDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	if db == nil {
		t.Fatal("Database instance is nil")
	}
}
