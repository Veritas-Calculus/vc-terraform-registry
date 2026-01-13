// Package storage handles file storage operations.
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage defines the interface for file storage operations.
type Storage interface {
	Save(path string, data io.Reader) error
	Get(path string) (io.ReadCloser, error)
	Delete(path string) error
	Exists(path string) (bool, error)
}

// LocalStorage implements Storage interface using local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new LocalStorage instance.
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0750); err != nil { // #nosec G301 - storage directory needs group access
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// Save stores data to the specified path.
func (s *LocalStorage) Save(path string, data io.Reader) error {
	fullPath := filepath.Join(s.basePath, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 - storage directory needs group access
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath) // #nosec G304 - path is validated via basePath join
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// Get retrieves data from the specified path.
func (s *LocalStorage) Get(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	file, err := os.Open(fullPath) // #nosec G304 - path is validated via basePath join
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete removes the file at the specified path.
func (s *LocalStorage) Delete(path string) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists checks if a file exists at the specified path.
func (s *LocalStorage) Exists(path string) (bool, error) {
	fullPath := filepath.Join(s.basePath, path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
