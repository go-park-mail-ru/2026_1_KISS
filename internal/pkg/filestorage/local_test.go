package filestorage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSave_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filestorage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := NewLocalStorage(tmpDir, "/uploads/")
	url, err := s.Save("test.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "/uploads/test.txt" {
		t.Errorf("expected /uploads/test.txt, got %s", url)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("expected file content 'hello', got %q", string(data))
	}
}

func TestSave_Subdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filestorage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := NewLocalStorage(tmpDir, "/uploads/")
	url, err := s.Save("datasets/data.csv", strings.NewReader("a,b,c"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "/uploads/datasets/data.csv" {
		t.Errorf("expected /uploads/datasets/data.csv, got %s", url)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "datasets", "data.csv"))
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(data) != "a,b,c" {
		t.Errorf("expected 'a,b,c', got %q", string(data))
	}
}

func TestSave_InvalidFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"dot-dot", "..evil.txt"},
		{"backslash", "sub\\file.txt"},
		{"only dots", ".."},
		{"deep path", "a/b/c.txt"},
		{"empty segment", "/file.txt"},
		{"trailing slash", "dir/"},
		{"empty", ""},
	}

	tmpDir, err := os.MkdirTemp("", "filestorage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := NewLocalStorage(tmpDir, "/uploads/")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Save(tt.filename, strings.NewReader("data"))
			if err == nil {
				t.Error("expected error for invalid filename")
			}
		})
	}
}

func TestDelete_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filestorage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := NewLocalStorage(tmpDir, "/uploads/")
	_, err = s.Save("to-delete.txt", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}

	err = s.Delete("/uploads/to-delete.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = os.Stat(filepath.Join(tmpDir, "to-delete.txt"))
	if !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestDelete_Subdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filestorage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := NewLocalStorage(tmpDir, "/uploads/")
	_, err = s.Save("datasets/file.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}

	err = s.Delete("/uploads/datasets/file.csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = os.Stat(filepath.Join(tmpDir, "datasets", "file.csv"))
	if !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestDelete_EmptyPath(t *testing.T) {
	s := NewLocalStorage("/tmp", "/uploads/")
	err := s.Delete("")
	if err != nil {
		t.Fatalf("expected nil error for empty path, got %v", err)
	}
}

func TestDelete_FileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filestorage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := NewLocalStorage(tmpDir, "/uploads/")
	err = s.Delete("/uploads/nonexistent.txt")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
}
