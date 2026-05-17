package filestorage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type LocalStorage struct {
	uploadDir string
	urlPrefix string
}

func NewLocalStorage(uploadDir, urlPrefix string) *LocalStorage {
	return &LocalStorage{
		uploadDir: uploadDir,
		urlPrefix: urlPrefix,
	}
}

func validateFilename(filename string) error {
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsRune(filename, '\\') {
		return fmt.Errorf("invalid filename")
	}

	parts := strings.Split(filename, "/")
	if len(parts) > 2 || slices.Contains(parts, "") {
		return fmt.Errorf("invalid filename")
	}
	return nil
}

func (s *LocalStorage) Save(filename string, data io.Reader) (string, error) {
	if err := validateFilename(filename); err != nil {
		return "", err
	}

	dst := filepath.Join(s.uploadDir, filename)
	dir := filepath.Dir(dst)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

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

func (s *LocalStorage) Open(storageKey string) (io.ReadCloser, error) {
	if err := validateFilename(storageKey); err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(s.uploadDir, storageKey))
}

func (s *LocalStorage) Delete(path string) error {
	if path == "" {
		return nil
	}

	filename := strings.TrimPrefix(path, s.urlPrefix)
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsRune(filename, '\\') {
		return nil
	}

	parts := strings.Split(filename, "/")
	if len(parts) > 2 || slices.Contains(parts, "") {
		return nil
	}

	dst := filepath.Join(s.uploadDir, filename)
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}
