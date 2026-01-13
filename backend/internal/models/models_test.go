package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSourceTypeConstants(t *testing.T) {
	t.Run("SourceUpload value", func(t *testing.T) {
		if SourceUpload != "upload" {
			t.Errorf("SourceUpload = %q, want %q", SourceUpload, "upload")
		}
	})

	t.Run("SourceMirror value", func(t *testing.T) {
		if SourceMirror != "mirror" {
			t.Errorf("SourceMirror = %q, want %q", SourceMirror, "mirror")
		}
	})
}

func TestProvider(t *testing.T) {
	t.Run("create provider struct", func(t *testing.T) {
		now := time.Now()
		provider := Provider{
			ID:          1,
			Namespace:   "hashicorp",
			Name:        "aws",
			Version:     "5.0.0",
			Description: "AWS Provider",
			SourceType:  SourceMirror,
			SourceURL:   "https://registry.terraform.io",
			Protocols:   `["5.0", "6.0"]`,
			Published:   now,
			Downloads:   1000,
		}

		if provider.Namespace != "hashicorp" {
			t.Errorf("Namespace = %q, want %q", provider.Namespace, "hashicorp")
		}
		if provider.Name != "aws" {
			t.Errorf("Name = %q, want %q", provider.Name, "aws")
		}
		if provider.Version != "5.0.0" {
			t.Errorf("Version = %q, want %q", provider.Version, "5.0.0")
		}
		if provider.SourceType != SourceMirror {
			t.Errorf("SourceType = %q, want %q", provider.SourceType, SourceMirror)
		}
	})

	t.Run("provider with platforms", func(t *testing.T) {
		provider := Provider{
			ID:        1,
			Namespace: "hashicorp",
			Name:      "aws",
			Version:   "5.0.0",
			Platforms: []ProviderPlatform{
				{ID: 1, ProviderID: 1, OS: "linux", Arch: "amd64"},
				{ID: 2, ProviderID: 1, OS: "darwin", Arch: "arm64"},
			},
		}

		if len(provider.Platforms) != 2 {
			t.Errorf("len(Platforms) = %d, want 2", len(provider.Platforms))
		}
		if provider.Platforms[0].OS != "linux" {
			t.Errorf("Platforms[0].OS = %q, want %q", provider.Platforms[0].OS, "linux")
		}
	})

	t.Run("provider JSON serialization", func(t *testing.T) {
		provider := Provider{
			ID:        1,
			Namespace: "hashicorp",
			Name:      "aws",
			Version:   "5.0.0",
		}

		data, err := json.Marshal(provider)
		if err != nil {
			t.Fatalf("json.Marshal error = %v", err)
		}

		var decoded Provider
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal error = %v", err)
		}

		if decoded.Namespace != provider.Namespace {
			t.Errorf("decoded.Namespace = %q, want %q", decoded.Namespace, provider.Namespace)
		}
	})
}

func TestModule(t *testing.T) {
	t.Run("create module struct", func(t *testing.T) {
		now := time.Now()
		module := Module{
			ID:          1,
			Namespace:   "hashicorp",
			Name:        "consul",
			Provider:    "aws",
			Version:     "1.0.0",
			Description: "Consul module for AWS",
			Source:      "github.com/hashicorp/terraform-aws-consul",
			Published:   now,
			Downloads:   500,
		}

		if module.Namespace != "hashicorp" {
			t.Errorf("Namespace = %q, want %q", module.Namespace, "hashicorp")
		}
		if module.Provider != "aws" {
			t.Errorf("Provider = %q, want %q", module.Provider, "aws")
		}
	})

	t.Run("module JSON serialization", func(t *testing.T) {
		module := Module{
			ID:        1,
			Namespace: "hashicorp",
			Name:      "consul",
			Provider:  "aws",
			Version:   "1.0.0",
		}

		data, err := json.Marshal(module)
		if err != nil {
			t.Fatalf("json.Marshal error = %v", err)
		}

		var decoded Module
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal error = %v", err)
		}

		if decoded.Name != module.Name {
			t.Errorf("decoded.Name = %q, want %q", decoded.Name, module.Name)
		}
	})
}

func TestUser(t *testing.T) {
	t.Run("create user struct", func(t *testing.T) {
		user := User{
			ID:       1,
			Username: "admin",
			Email:    "admin@example.com",
			Password: "hashed_password",
			Role:     "admin",
			APIToken: "token123",
		}

		if user.Username != "admin" {
			t.Errorf("Username = %q, want %q", user.Username, "admin")
		}
		if user.Role != "admin" {
			t.Errorf("Role = %q, want %q", user.Role, "admin")
		}
	})

	t.Run("user password not in JSON", func(t *testing.T) {
		user := User{
			ID:       1,
			Username: "admin",
			Email:    "admin@example.com",
			Password: "secret_password",
			Role:     "admin",
		}

		data, err := json.Marshal(user)
		if err != nil {
			t.Fatalf("json.Marshal error = %v", err)
		}

		// Password should not be in JSON output (has json:"-" tag)
		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal error = %v", err)
		}

		if _, exists := decoded["password"]; exists {
			t.Error("password should not be in JSON output")
		}
	})
}

func TestProviderPlatform(t *testing.T) {
	t.Run("create provider platform", func(t *testing.T) {
		platform := ProviderPlatform{
			ID:         1,
			ProviderID: 1,
			OS:         "linux",
			Arch:       "amd64",
			Filename:   "terraform-provider-aws_5.0.0_linux_amd64.zip",
			FilePath:   "/data/providers/hashicorp/aws/5.0.0/linux_amd64/terraform-provider-aws_5.0.0_linux_amd64.zip",
			SHA256Sum:  "abc123def456",
			FileSize:   1024000,
		}

		if platform.OS != "linux" {
			t.Errorf("OS = %q, want %q", platform.OS, "linux")
		}
		if platform.Arch != "amd64" {
			t.Errorf("Arch = %q, want %q", platform.Arch, "amd64")
		}
		if platform.FileSize != 1024000 {
			t.Errorf("FileSize = %d, want 1024000", platform.FileSize)
		}
	})
}

func TestMirrorConfig(t *testing.T) {
	t.Run("create mirror config", func(t *testing.T) {
		now := time.Now()
		config := MirrorConfig{
			ID:          1,
			Namespace:   "hashicorp",
			Name:        "aws",
			UpstreamURL: "https://registry.terraform.io",
			AutoSync:    true,
			LastSyncAt:  &now,
		}

		if config.Namespace != "hashicorp" {
			t.Errorf("Namespace = %q, want %q", config.Namespace, "hashicorp")
		}
		if !config.AutoSync {
			t.Error("AutoSync should be true")
		}
		if config.LastSyncAt == nil {
			t.Error("LastSyncAt should not be nil")
		}
	})

	t.Run("mirror config with nil LastSyncAt", func(t *testing.T) {
		config := MirrorConfig{
			ID:          1,
			Namespace:   "hashicorp",
			Name:        "aws",
			UpstreamURL: "https://registry.terraform.io",
			AutoSync:    false,
			LastSyncAt:  nil,
		}

		if config.LastSyncAt != nil {
			t.Error("LastSyncAt should be nil")
		}
	})
}

func TestSettings(t *testing.T) {
	t.Run("create settings with defaults", func(t *testing.T) {
		settings := Settings{
			ID:                 1,
			AllowOnlineSearch:  true,
			DefaultUpstreamURL: "https://registry.terraform.io",
			RegistryURL:        "https://my-registry.example.com",
			ProxyEnabled:       false,
			ProxyURL:           "",
			ProxyType:          "http",
		}

		if !settings.AllowOnlineSearch {
			t.Error("AllowOnlineSearch should be true")
		}
		if settings.DefaultUpstreamURL != "https://registry.terraform.io" {
			t.Errorf("DefaultUpstreamURL = %q, want %q", settings.DefaultUpstreamURL, "https://registry.terraform.io")
		}
		if settings.ProxyEnabled {
			t.Error("ProxyEnabled should be false")
		}
	})

	t.Run("settings with proxy enabled", func(t *testing.T) {
		settings := Settings{
			ID:           1,
			ProxyEnabled: true,
			ProxyURL:     "socks5://127.0.0.1:1080",
			ProxyType:    "socks5",
		}

		if !settings.ProxyEnabled {
			t.Error("ProxyEnabled should be true")
		}
		if settings.ProxyType != "socks5" {
			t.Errorf("ProxyType = %q, want %q", settings.ProxyType, "socks5")
		}
	})
}

func TestSyncSchedule(t *testing.T) {
	t.Run("create sync schedule", func(t *testing.T) {
		now := time.Now()
		nextRun := now.Add(time.Hour)
		schedule := SyncSchedule{
			ID:         1,
			Namespace:  "hashicorp",
			Name:       "aws",
			CronExpr:   "0 0 * * *",
			Enabled:    true,
			SyncOS:     "all",
			SyncArch:   "all",
			LastRunAt:  &now,
			LastStatus: "success",
			LastError:  "",
			NextRunAt:  &nextRun,
		}

		if schedule.CronExpr != "0 0 * * *" {
			t.Errorf("CronExpr = %q, want %q", schedule.CronExpr, "0 0 * * *")
		}
		if !schedule.Enabled {
			t.Error("Enabled should be true")
		}
		if schedule.LastStatus != "success" {
			t.Errorf("LastStatus = %q, want %q", schedule.LastStatus, "success")
		}
	})

	t.Run("sync schedule with specific platform", func(t *testing.T) {
		schedule := SyncSchedule{
			ID:        1,
			Namespace: "hashicorp",
			Name:      "aws",
			CronExpr:  "0 */6 * * *",
			Enabled:   true,
			SyncOS:    "linux",
			SyncArch:  "amd64",
		}

		if schedule.SyncOS != "linux" {
			t.Errorf("SyncOS = %q, want %q", schedule.SyncOS, "linux")
		}
		if schedule.SyncArch != "amd64" {
			t.Errorf("SyncArch = %q, want %q", schedule.SyncArch, "amd64")
		}
	})

	t.Run("sync schedule with failed status", func(t *testing.T) {
		schedule := SyncSchedule{
			ID:         1,
			Namespace:  "hashicorp",
			Name:       "nonexistent",
			CronExpr:   "0 0 * * *",
			Enabled:    true,
			LastStatus: "failed",
			LastError:  "provider not found",
		}

		if schedule.LastStatus != "failed" {
			t.Errorf("LastStatus = %q, want %q", schedule.LastStatus, "failed")
		}
		if schedule.LastError == "" {
			t.Error("LastError should not be empty for failed sync")
		}
	})
}
