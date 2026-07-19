// Package logsafe renders untrusted values safe to embed in log records.
package logsafe

import (
	"errors"
	"strings"
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain identifier unchanged", "hashicorp", "hashicorp"},
		{"identifier with separators unchanged", "my-org_1", "my-org_1"},
		{"spaces are preserved", "a b c", "a b c"},
		{"empty string", "", ""},
		{"newline becomes space", "a\nb", "a b"},
		{"carriage return becomes space", "a\rb", "a b"},
		{"crlf becomes two spaces", "a\r\nb", "a  b"},
		{"forged log line neutralized", "ok\nINFO: fake entry", "ok INFO: fake entry"},
		{"ansi escape stripped", "a\x1b[2Jb", "a[2Jb"},
		{"osc title escape stripped", "a\x1b]0;pwn\x07b", "a]0;pwnb"},
		{"nul stripped", "a\x00b", "ab"},
		{"tab stripped", "a\tb", "ab"},
		{"line separator stripped", "a\u2028b", "ab"},
		{"paragraph separator stripped", "a\u2029b", "ab"},
		{"bidi override stripped", "a\u202eb", "ab"},
		{"unicode preserved", "café", "café"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Clean(tt.in); got != tt.want {
				t.Errorf("Clean(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCleanTruncates(t *testing.T) {
	t.Run("long input is truncated", func(t *testing.T) {
		got := Clean(strings.Repeat("a", maxLen+50))
		if !strings.HasSuffix(got, truncationSuffix) {
			t.Errorf("Clean(long) = %q, want the %q suffix", got, truncationSuffix)
		}
		if len(got) != maxLen+len(truncationSuffix) {
			t.Errorf("len(Clean(long)) = %d, want %d", len(got), maxLen+len(truncationSuffix))
		}
	})

	t.Run("input at the limit is untouched", func(t *testing.T) {
		in := strings.Repeat("a", maxLen)
		if got := Clean(in); got != in {
			t.Errorf("Clean(exactly maxLen) = %q, want it unchanged", got)
		}
	})

	t.Run("multibyte input truncates on a rune boundary", func(t *testing.T) {
		got := Clean(strings.Repeat("é", maxLen))
		if strings.Contains(got, "\uFFFD") {
			t.Errorf("Clean(multibyte) = %q, want no replacement character", got)
		}
	})
}

func TestCleanErr(t *testing.T) {
	t.Run("nil error returns empty string", func(t *testing.T) {
		if got := CleanErr(nil); got != "" {
			t.Errorf("CleanErr(nil) = %q, want %q", got, "")
		}
	})

	t.Run("error message is cleaned", func(t *testing.T) {
		err := errors.New("upstream failed\nINFO: forged=entry")
		want := "upstream failed INFO: forged=entry"
		if got := CleanErr(err); got != want {
			t.Errorf("CleanErr(err) = %q, want %q", got, want)
		}
	})
}
