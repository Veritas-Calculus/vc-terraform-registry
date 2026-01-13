// Package storage handles file storage operations.
package storage

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLocalStorage(t *testing.T) {
	t.Run("create with valid path", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		storagePath := filepath.Join(tempDir, "storage")
		storage, err := NewLocalStorage(storagePath)
		if err != nil {
			t.Fatalf("NewLocalStorage() error = %v", err)
		}
		if storage == nil {
			t.Error("NewLocalStorage() returned nil")
		}

		if _, err := os.Stat(storagePath); os.IsNotExist(err) {
			t.Error("NewLocalStorage() did not create directory")
		}
	})

	t.Run("create with nested path", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		storagePath := filepath.Join(tempDir, "a", "b", "c", "storage")
		storage, err := NewLocalStorage(storagePath)
		if err != nil {
			t.Fatalf("NewLocalStorage() error = %v", err)
		}
		if storage == nil {
			t.Error("NewLocalStorage() returned nil")
		}
	})
}

func TestLocalStorage_Save(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	storage, _ := NewLocalStorage(tempDir)

	t.Run("save file successfully", func(t *testing.T) {
		data := bytes.NewReader([]byte("test content"))
		err := storage.Save("test.txt", data)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		fullPath := filepath.Join(tempDir, "test.txt")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Error("Save() did not create file")
		}

		content, _ := os.ReadFile(fullPath) //nolint:gosec // test file path is safe
		if string(content) != "test content" {
			t.Errorf("file content = %q, want %q", string(content), "test content")
		}
	})

	t.Run("save file in nested directory", func(t *testing.T) {
		data := bytes.NewReader([]byte("nested content"))
		err := storage.Save("a/b/c/nested.txt", data)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		fullPath := filepath.Join(tempDir, "a", "b", "c", "nested.txt")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Error("Save() did not create nested file")
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		_ = storage.Save("overwrite.txt", bytes.NewReader([]byte("original")))

		err := storage.Save("overwrite.txt", bytes.NewReader([]byte("updated")))
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		fullPath := filepath.Join(tempDir, "overwrite.txt")
		content, _ := os.ReadFile(fullPath) //nolint:gosec // test file path is safe
		if string(content) != "updated" {
			t.Errorf("file content = %q, want %q", string(content), "updated")
		}
	})

	t.Run("save empty file", func(t *testing.T) {
		data := bytes.NewReader([]byte{})
		err := storage.Save("empty.txt", data)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		fullPath := filepath.Join(tempDir, "empty.txt")
		info, _ := os.Stat(fullPath)
		if info.Size() != 0 {
			t.Errorf("empty file size = %d, want 0", info.Size())
		}
	})
}

func TestLocalStorage_Get(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	storage, _ := NewLocalStorage(tempDir)

	t.Run("get existing file", func(t *testing.T) {
		testContent := "hello world"
		_ = storage.Save("gettest.txt", bytes.NewReader([]byte(testContent)))

		reader, err := storage.Get("gettest.txt")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		defer func() { _ = reader.Close() }()

		content, _ := io.ReadAll(reader)
		if string(content) != testContent {
			t.Errorf("Get() content = %q, want %q", string(content), testContent)
		}
	})

	t.Run("get non-existing file", func(t *testing.T) {
		_, err := storage.Get("nonexistent.txt")
		if err == nil {
			t.Error("Get() expected error for non-existing file")
		}
	})

	t.Run("get file from nested directory", func(t *testing.T) {
		_ = storage.Save("nested/path/file.txt", bytes.NewReader([]byte("nested")))

		reader, err := storage.Get("nested/path/file.txt")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		defer func() { _ = reader.Close() }()

		content, _ := io.ReadAll(reader)
		if string(content) != "nested" {
			t.Errorf("Get() content = %q, want %q", string(content), "nested")
		}
	})
}

func TestLocalStorage_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	storage, _ := NewLocalStorage(tempDir)

	t.Run("delete existing file", func(t *testing.T) {
		_ = storage.Save("todelete.txt", bytes.NewReader([]byte("delete me")))

		err := storage.Delete("todelete.txt")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		fullPath := filepath.Join(tempDir, "todelete.txt")
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Error("Delete() did not remove file")
		}
	})

	t.Run("delete non-existing file", func(t *testing.T) {
		err := storage.Delete("nonexistent.txt")
		if err == nil {
			t.Error("Delete() expected error for non-existing file")
		}
	})
}

func TestLocalStorage_Exists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	storage, _ := NewLocalStorage(tempDir)

	t.Run("exists for existing file", func(t *testing.T) {
		_ = storage.Save("exists.txt", bytes.NewReader([]byte("content")))

		exists, err := storage.Exists("exists.txt")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() = false for existing file")
		}
	})

	t.Run("exists for non-existing file", func(t *testing.T) {
		exists, err := storage.Exists("nonexistent.txt")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() = true for non-existing file")
		}
	})

	t.Run("exists after delete", func(t *testing.T) {
		_ = storage.Save("willdelete.txt", bytes.NewReader([]byte("content")))
		_ = storage.Delete("willdelete.txt")

		exists, err := storage.Exists("willdelete.txt")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() = true for deleted file")
		}
	})
}

func TestLocalStorage_IntegrationWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	storage, _ := NewLocalStorage(tempDir)

	path := "providers/hashicorp/aws/5.0.0/linux_amd64/terraform-provider-aws"

	// 1. Check doesn't exist initially
	exists, _ := storage.Exists(path)
	if exists {
		t.Error("file should not exist initially")
	}

	// 2. Save file
	content := "provider binary content"
	err = storage.Save(path, bytes.NewReader([]byte(content)))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 3. Check exists
	exists, _ = storage.Exists(path)
	if !exists {
		t.Error("file should exist after save")
	}

	// 4. Get and verify content
	reader, _ := storage.Get(path)
	retrieved, _ := io.ReadAll(reader)
	_ = reader.Close()
	if string(retrieved) != content {
		t.Errorf("content = %q, want %q", string(retrieved), content)
	}

	// 5. Delete
	err = storage.Delete(path)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 6. Verify deleted
	exists, _ = storage.Exists(path)
	if exists {
		t.Error("file should not exist after delete")
	}
}
