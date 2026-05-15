package filevalidate_test

import (
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/filevalidate"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name     string
		category domain.FileCategory
		filename string
		mime     string
		wantErr  bool
	}{
		{"png ok", domain.FileCategoryGeneral, "pic.png", "image/png", false},
		{"jpg compat with image/jpeg", domain.FileCategoryGeneral, "pic.jpg", "image/jpeg", false},
		{"jpeg compat with image/jpeg", domain.FileCategoryGeneral, "pic.jpeg", "image/jpeg", false},
		{"pdf ok", domain.FileCategoryGeneral, "doc.pdf", "application/pdf", false},
		{"txt ok", domain.FileCategoryGeneral, "notes.txt", "text/plain", false},
		{"csv as text/plain ok", domain.FileCategoryGeneral, "data.csv", "text/plain", false},
		{"json ok", domain.FileCategoryGeneral, "x.json", "application/json", false},
		{"zip ok", domain.FileCategoryGeneral, "a.zip", "application/zip", false},
		{"docx as zip ok", domain.FileCategoryGeneral, "doc.docx", "application/zip", false},

		{"avatar png ok", domain.FileCategoryAvatar, "a.png", "image/png", false},
		{"avatar pdf rejected", domain.FileCategoryAvatar, "a.pdf", "application/pdf", true},

		{"exe blocked", domain.FileCategoryGeneral, "evil.exe", "application/octet-stream", true},
		{"bat blocked", domain.FileCategoryGeneral, "evil.bat", "text/plain", true},
		{"sh blocked", domain.FileCategoryGeneral, "run.sh", "text/plain", true},

		{"mime not in whitelist", domain.FileCategoryGeneral, "a.bin", "application/x-msdownload", true},

		{"png renamed to jpg mismatch", domain.FileCategoryGeneral, "fake.png", "image/jpeg", true},
		{"pdf renamed to png mismatch", domain.FileCategoryGeneral, "fake.png", "application/pdf", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := filevalidate.Validate(c.category, c.filename, c.mime)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
