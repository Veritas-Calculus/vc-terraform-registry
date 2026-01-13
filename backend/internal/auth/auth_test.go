// Package auth provides authentication and authorization functionality.
package auth

import (
	"testing"
	"time"
)

func TestJWTManager_Generate(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Hour)

	t.Run("generate valid token", func(t *testing.T) {
		token, err := manager.Generate(1, "testuser", "admin")
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if token == "" {
			t.Error("Generate() returned empty token")
		}
	})

	t.Run("generate token with different users", func(t *testing.T) {
		token1, _ := manager.Generate(1, "user1", "admin")
		token2, _ := manager.Generate(2, "user2", "user")
		if token1 == token2 {
			t.Error("tokens for different users should be different")
		}
	})
}

func TestJWTManager_Verify(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Hour)

	t.Run("verify valid token", func(t *testing.T) {
		token, err := manager.Generate(123, "testuser", "admin")
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		claims, err := manager.Verify(token)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}

		if claims.UserID != 123 {
			t.Errorf("claims.UserID = %d, want 123", claims.UserID)
		}
		if claims.Username != "testuser" {
			t.Errorf("claims.Username = %q, want %q", claims.Username, "testuser")
		}
		if claims.Role != "admin" {
			t.Errorf("claims.Role = %q, want %q", claims.Role, "admin")
		}
	})

	t.Run("verify invalid token", func(t *testing.T) {
		_, err := manager.Verify("invalid-token")
		if err != ErrInvalidToken {
			t.Errorf("Verify() error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("verify empty token", func(t *testing.T) {
		_, err := manager.Verify("")
		if err != ErrInvalidToken {
			t.Errorf("Verify() error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("verify token with wrong secret", func(t *testing.T) {
		token, _ := manager.Generate(1, "user", "admin")
		wrongManager := NewJWTManager("wrong-secret", time.Hour)
		_, err := wrongManager.Verify(token)
		if err != ErrInvalidToken {
			t.Errorf("Verify() error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("verify expired token", func(t *testing.T) {
		shortManager := NewJWTManager("test-secret", time.Nanosecond)
		token, _ := shortManager.Generate(1, "user", "admin")
		time.Sleep(time.Millisecond * 10)
		_, err := shortManager.Verify(token)
		if err != ErrExpiredToken && err != ErrInvalidToken {
			t.Errorf("Verify() error = %v, want ErrExpiredToken or ErrInvalidToken", err)
		}
	})
}

func TestHashPassword(t *testing.T) {
	t.Run("hash password successfully", func(t *testing.T) {
		hash, err := HashPassword("mysecretpassword")
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}
		if hash == "" {
			t.Error("HashPassword() returned empty hash")
		}
		if hash == "mysecretpassword" {
			t.Error("HashPassword() should not return plaintext password")
		}
	})

	t.Run("same password produces different hashes", func(t *testing.T) {
		hash1, _ := HashPassword("password123")
		hash2, _ := HashPassword("password123")
		if hash1 == hash2 {
			t.Error("HashPassword() should produce different hashes for same password")
		}
	})

	t.Run("different passwords produce different hashes", func(t *testing.T) {
		hash1, _ := HashPassword("password1")
		hash2, _ := HashPassword("password2")
		if hash1 == hash2 {
			t.Error("different passwords should produce different hashes")
		}
	})
}

func TestCheckPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, _ := HashPassword(password)

	t.Run("correct password", func(t *testing.T) {
		if !CheckPassword(password, hash) {
			t.Error("CheckPassword() = false for correct password")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		if CheckPassword("wrongpassword", hash) {
			t.Error("CheckPassword() = true for wrong password")
		}
	})

	t.Run("empty password against hash", func(t *testing.T) {
		if CheckPassword("", hash) {
			t.Error("CheckPassword() = true for empty password against non-empty hash")
		}
	})

	t.Run("correct password against invalid hash", func(t *testing.T) {
		if CheckPassword(password, "invalid-hash") {
			t.Error("CheckPassword() = true for invalid hash")
		}
	})
}

func TestNewJWTManager(t *testing.T) {
	t.Run("create with valid parameters", func(t *testing.T) {
		manager := NewJWTManager("secret", 24*time.Hour)
		if manager == nil {
			t.Error("NewJWTManager() returned nil")
		}
	})

	t.Run("create with empty secret", func(t *testing.T) {
		manager := NewJWTManager("", time.Hour)
		if manager == nil {
			t.Error("NewJWTManager() returned nil")
		}
		token, err := manager.Generate(1, "user", "admin")
		if err != nil {
			t.Errorf("Generate() with empty secret error = %v", err)
		}
		if token == "" {
			t.Error("Generate() with empty secret returned empty token")
		}
	})
}

func TestClaims(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	token, _ := manager.Generate(42, "admin", "superuser")

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	t.Run("check claims fields", func(t *testing.T) {
		if claims.UserID != 42 {
			t.Errorf("UserID = %d, want 42", claims.UserID)
		}
		if claims.Username != "admin" {
			t.Errorf("Username = %q, want %q", claims.Username, "admin")
		}
		if claims.Role != "superuser" {
			t.Errorf("Role = %q, want %q", claims.Role, "superuser")
		}
	})

	t.Run("check registered claims", func(t *testing.T) {
		if claims.ExpiresAt == nil {
			t.Error("ExpiresAt should not be nil")
		}
		if claims.IssuedAt == nil {
			t.Error("IssuedAt should not be nil")
		}
	})
}
