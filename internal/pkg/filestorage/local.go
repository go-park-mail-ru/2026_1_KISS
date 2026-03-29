package filestorage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	uploadDir string
	urlPrefix string
}

// NewLocalStorage creates a new LocalStorage instance.
func NewLocalStorage(uploadDir, urlPrefix string) *LocalStorage {
	return &LocalStorage{
		uploadDir: uploadDir,
		urlPrefix: urlPrefix,
	}
}

// Save writes data to uploadDir/filename and returns the public URL path.
func (s *LocalStorage) Save(filename string, data io.Reader) (string, error) {
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, '/') || strings.ContainsRune(filename, '\\') {
		return "", fmt.Errorf("invalid filename")
	}

	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

	dst := filepath.Join(s.uploadDir, filename)
	f, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		os.Remove(dst)
		return "", fmt.Errorf("write file: %w", err)
	}

	return s.urlPrefix + filename, nil
}

// Delete removes a file by its URL path. Ignores "file not found" errors.
func (s *LocalStorage) Delete(path string) error {
	if path == "" {
		return nil
	}

	filename := strings.TrimPrefix(path, s.urlPrefix)
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsRune(filename, '/') {
		return nil
	}

	dst := filepath.Join(s.uploadDir, filename)
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}
