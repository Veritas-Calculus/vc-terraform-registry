// Package api provides HTTP handlers for the Provider Mirror Protocol.
package api

import (
	"testing"
)

func TestValidateProviderParams(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		provider  string
		version   string
		wantError bool
	}{
		{"valid params", "hashicorp", "aws", "5.0.0", false},
		{"valid params with prerelease", "hashicorp", "aws", "5.0.0-beta1", false},
		{"valid params with build metadata", "hashicorp", "aws", "5.0.0+build.123", false},
		{"valid params with hyphen in name", "my-org", "my-provider", "1.0.0", false},
		{"valid params with underscore", "my_org", "my_provider", "1.0.0", false},
		{"valid params without version", "hashicorp", "aws", "", false},
		{"invalid namespace - uppercase", "HashiCorp", "aws", "5.0.0", true},
		{"invalid namespace - special chars", "hashi@corp", "aws", "5.0.0", true},
		{"invalid namespace - path traversal", "../etc", "passwd", "1.0.0", true},
		{"invalid namespace - too long", "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz01234", "aws", "1.0.0", true},
		{"invalid namespace - newline injection", "valid\nINFO: fake log entry", "aws", "1.0.0", true},
		{"invalid provider - starts with hyphen", "hashicorp", "-aws", "1.0.0", true},
		{"invalid version - not semver", "hashicorp", "aws", "5.0", true},
		{"invalid version - text", "hashicorp", "aws", "latest", true},
		{"invalid version - injection attempt", "hashicorp", "aws", "1.0.0\nmalicious", true},
		{"empty namespace", "", "aws", "1.0.0", true},
		{"empty provider", "hashicorp", "", "1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := validateProviderParams(tt.namespace, tt.provider, tt.version)
			if tt.wantError && errMsg == "" {
				t.Errorf("validateProviderParams(%q, %q, %q) expected error, got empty string",
					tt.namespace, tt.provider, tt.version)
			}
			if !tt.wantError && errMsg != "" {
				t.Errorf("validateProviderParams(%q, %q, %q) unexpected error: %s",
					tt.namespace, tt.provider, tt.version, errMsg)
			}
		})
	}
}

func TestValidatePlatform(t *testing.T) {
	tests := []struct {
		name     string
		os       string
		arch     string
		wantOS   string
		wantArch string
	}{
		{"valid linux amd64", "linux", "amd64", "linux", "amd64"},
		{"valid darwin arm64", "darwin", "arm64", "darwin", "arm64"},
		{"valid windows 386", "windows", "386", "windows", "386"},
		{"invalid os with uppercase", "Linux", "amd64", "invalid", "amd64"},
		{"invalid arch with special chars", "linux", "amd64\nmalicious", "linux", "invalid"},
		{"both invalid", "../etc", "@#$%", "invalid", "invalid"},
		{"empty values", "", "", "invalid", "invalid"},
		{"too long os", "abcdefghijklmnopqrstuvwxyz0123456", "amd64", "invalid", "amd64"},
		{"os starts with hyphen", "-linux", "amd64", "invalid", "amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOS, gotArch := validatePlatform(tt.os, tt.arch)
			if gotOS != tt.wantOS {
				t.Errorf("validatePlatform(%q, %q) OS = %q, want %q",
					tt.os, tt.arch, gotOS, tt.wantOS)
			}
			if gotArch != tt.wantArch {
				t.Errorf("validatePlatform(%q, %q) Arch = %q, want %q",
					tt.os, tt.arch, gotArch, tt.wantArch)
			}
		})
	}
}

func TestValidIdentifierRegex(t *testing.T) {
	validCases := []string{"hashicorp", "aws", "my-provider", "my_provider", "provider123", "a", "1provider"}
	invalidCases := []string{"", "-provider", "_provider", "HashiCorp", "has space", "has@symbol", "has/slash"}

	for _, tc := range validCases {
		if !validIdentifier.MatchString(tc) {
			t.Errorf("validIdentifier should match %q", tc)
		}
	}

	for _, tc := range invalidCases {
		if validIdentifier.MatchString(tc) {
			t.Errorf("validIdentifier should not match %q", tc)
		}
	}
}

func TestValidVersionRegex(t *testing.T) {
	validCases := []string{"1.0.0", "0.0.1", "10.20.30", "1.0.0-alpha", "1.0.0-alpha.1", "1.0.0+build", "1.0.0-rc.1+build.456"}
	invalidCases := []string{"", "1", "1.0", "v1.0.0", "1.0.0.0", "latest", "1.0.0-", "1.0.0+"}

	for _, tc := range validCases {
		if !validVersion.MatchString(tc) {
			t.Errorf("validVersion should match %q", tc)
		}
	}

	for _, tc := range invalidCases {
		if validVersion.MatchString(tc) {
			t.Errorf("validVersion should not match %q", tc)
		}
	}
}

func TestValidIdentifierStrictRegex(t *testing.T) {
	validCases := []string{"linux", "darwin", "windows", "amd64", "arm64", "386", "freebsd"}
	invalidCases := []string{"", "Linux", "-linux"}

	for _, tc := range validCases {
		if !validIdentifierStrict.MatchString(tc) {
			t.Errorf("validIdentifierStrict should match %q", tc)
		}
	}

	for _, tc := range invalidCases {
		if validIdentifierStrict.MatchString(tc) {
			t.Errorf("validIdentifierStrict should not match %q", tc)
		}
	}
}
