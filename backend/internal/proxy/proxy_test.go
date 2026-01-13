// Package proxy provides functionality to mirror providers from upstream registries.
package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizePathComponent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{"valid simple name", "hashicorp", "hashicorp", false},
		{"valid name with hyphen", "my-provider", "my-provider", false},
		{"valid name with underscore", "my_provider", "my_provider", false},
		{"valid name with dot", "provider.v1", "provider.v1", false},
		{"valid version", "1.0.0", "1.0.0", false},
		{"empty string", "", "", true},
		{"path traversal with double dots", "..", "", true},
		{"path traversal in middle", "foo/../bar", "", true},
		{"forward slash", "foo/bar", "", true},
		{"backslash", "foo\\bar", "", true},
		{"null byte", "foo\x00bar", "", true},
		{"starts with hyphen", "-invalid", "", true},
		{"starts with dot", ".hidden", "", true},
		{"special characters", "foo@bar", "", true},
		{"space in name", "foo bar", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizePathComponent(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("SanitizePathComponent(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("SanitizePathComponent(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("SanitizePathComponent(%q) = %q, want %q", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestBuildSafeProviderPath(t *testing.T) {
	storagePath := "/data/providers"

	tests := []struct {
		name      string
		namespace string
		provider  string
		version   string
		osType    string
		arch      string
		wantError bool
		wantPath  string
	}{
		{
			name:      "valid path",
			namespace: "hashicorp",
			provider:  "aws",
			version:   "5.0.0",
			osType:    "linux",
			arch:      "amd64",
			wantError: false,
			wantPath:  filepath.Join(storagePath, "hashicorp", "aws", "5.0.0", "linux", "amd64"),
		},
		{
			name:      "valid path with prerelease version",
			namespace: "hashicorp",
			provider:  "aws",
			version:   "5.0.0-beta1",
			osType:    "darwin",
			arch:      "arm64",
			wantError: false,
			wantPath:  filepath.Join(storagePath, "hashicorp", "aws", "5.0.0-beta1", "darwin", "arm64"),
		},
		{
			name:      "invalid namespace with path traversal",
			namespace: "../etc",
			provider:  "aws",
			version:   "5.0.0",
			osType:    "linux",
			arch:      "amd64",
			wantError: true,
		},
		{
			name:      "invalid provider name",
			namespace: "hashicorp",
			provider:  "aws/../../etc",
			version:   "5.0.0",
			osType:    "linux",
			arch:      "amd64",
			wantError: true,
		},
		{
			name:      "empty namespace",
			namespace: "",
			provider:  "aws",
			version:   "5.0.0",
			osType:    "linux",
			arch:      "amd64",
			wantError: true,
		},
		{
			name:      "null byte in arch",
			namespace: "hashicorp",
			provider:  "aws",
			version:   "5.0.0",
			osType:    "linux",
			arch:      "amd64\x00evil",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSafeProviderPath(storagePath, tt.namespace, tt.provider, tt.version, tt.osType, tt.arch)
			if tt.wantError {
				if err == nil {
					t.Errorf("BuildSafeProviderPath() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("BuildSafeProviderPath() unexpected error: %v", err)
				}
				if got != tt.wantPath {
					t.Errorf("BuildSafeProviderPath() = %q, want %q", got, tt.wantPath)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{"simple filename", "terraform-provider-aws_5.0.0_linux_amd64.zip", "terraform-provider-aws_5.0.0_linux_amd64.zip", false},
		{"filename with path prefix stripped", "/some/path/file.zip", "file.zip", false},
		{"empty filename", "", "", true},
		{"dot only", ".", "", true},
		{"double dot", "..", "", true},
		{"null byte in filename", "file\x00.zip", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeFilename(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("SanitizeFilename(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("SanitizeFilename(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestNewProxyService(t *testing.T) {
	storagePath := "/tmp/test-storage"

	t.Run("with default upstream", func(t *testing.T) {
		ps := NewProxyService(storagePath, "")
		if ps == nil {
			t.Fatal("NewProxyService returned nil")
		}
		if ps.storagePath != storagePath {
			t.Errorf("storagePath = %q, want %q", ps.storagePath, storagePath)
		}
		if ps.upstreamURL != UpstreamRegistry {
			t.Errorf("upstreamURL = %q, want %q", ps.upstreamURL, UpstreamRegistry)
		}
	})

	t.Run("with custom upstream", func(t *testing.T) {
		customURL := "https://custom.registry.io"
		ps := NewProxyService(storagePath, customURL)
		if ps.upstreamURL != customURL {
			t.Errorf("upstreamURL = %q, want %q", ps.upstreamURL, customURL)
		}
	})
}

func TestProxyService_SetProxy(t *testing.T) {
	ps := NewProxyService("/tmp/test", "")

	t.Run("enable SOCKS5 proxy", func(t *testing.T) {
		ps.SetProxy(true, "socks5://127.0.0.1:1080", "socks5")
		ps.mu.RLock()
		defer ps.mu.RUnlock()
		if !ps.proxyEnabled {
			t.Error("proxy should be enabled")
		}
		if ps.proxyURL != "socks5://127.0.0.1:1080" {
			t.Errorf("proxyURL = %q, want %q", ps.proxyURL, "socks5://127.0.0.1:1080")
		}
	})

	t.Run("disable proxy", func(t *testing.T) {
		ps.SetProxy(false, "", "")
		ps.mu.RLock()
		defer ps.mu.RUnlock()
		if ps.proxyEnabled {
			t.Error("proxy should be disabled")
		}
	})
}

func TestProxyService_DownloadAndStoreProvider(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "proxy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	ps := NewProxyService(tempDir, "")

	t.Run("path traversal attack prevention", func(t *testing.T) {
		_, _, err := ps.DownloadAndStoreProvider(
			"../../../etc",
			"passwd",
			"1.0.0",
			"linux",
			"amd64",
			"http://example.com/file.zip",
		)
		if err == nil {
			t.Error("expected error for path traversal attack, got nil")
		}
		if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("expected 'invalid' in error message, got: %v", err)
		}
	})
}
