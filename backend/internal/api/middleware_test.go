package api

import (
	"strings"
	"testing"
	"time"
)

func TestRedactQuery(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/api/v1/mirror/x/y/stream", "/api/v1/mirror/x/y/stream"},
		{"/mirror/export/1?token=eyJhbGci: secret", ""},
	}
	// First case: no query, unchanged.
	if got := redactQuery(cases[0].in); got != cases[0].want {
		t.Errorf("redactQuery(%q) = %q, want %q", cases[0].in, got, cases[0].want)
	}
	// Token must never survive in the output.
	got := redactQuery("/mirror/export/1?token=eyJsecretJWT&os=linux")
	if strings.Contains(got, "eyJsecretJWT") {
		t.Errorf("redactQuery leaked the token: %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Errorf("redactQuery did not mark the token REDACTED: %q", got)
	}
	if !strings.Contains(got, "os=linux") {
		t.Errorf("redactQuery dropped a non-sensitive param: %q", got)
	}
}

func TestRateLimiter(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rl := newRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4", base) {
			t.Fatalf("request %d should be allowed within the limit", i+1)
		}
	}
	if rl.allow("1.2.3.4", base) {
		t.Fatal("4th request in the window should be blocked")
	}
	// A different key is independent.
	if !rl.allow("5.6.7.8", base) {
		t.Fatal("a different client should not be throttled")
	}
	// After the window elapses, the original key is allowed again.
	if !rl.allow("1.2.3.4", base.Add(time.Minute+time.Second)) {
		t.Fatal("request after the window should be allowed")
	}
}
