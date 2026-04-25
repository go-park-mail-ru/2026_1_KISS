package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestSession_IsExpired_True(t *testing.T) {
	s := &domain.Session{ExpiresAt: time.Now().Add(-time.Hour)}
	if !s.IsExpired() {
		t.Error("want expired")
	}
}

func TestSession_IsExpired_False(t *testing.T) {
	s := &domain.Session{ExpiresAt: time.Now().Add(time.Hour)}
	if s.IsExpired() {
		t.Error("want not expired")
	}
}

func TestValidFileCategory(t *testing.T) {
	valid := []domain.FileCategory{
		domain.FileCategoryAvatar,
		domain.FileCategoryFeedback,
		domain.FileCategoryDataset,
		domain.FileCategoryGeneral,
	}
	for _, c := range valid {
		if !domain.ValidFileCategory(c) {
			t.Errorf("expected %s to be valid", c)
		}
	}
	if domain.ValidFileCategory("invalid") {
		t.Error("expected 'invalid' to be invalid category")
	}
}

func TestErrors(t *testing.T) {
	errs := []error{
		domain.ErrNotFound,
		domain.ErrUnauthorized,
		domain.ErrConflict,
		domain.ErrInvalidInput,
		domain.ErrForbidden,
	}
	for _, e := range errs {
		if e == nil {
			t.Error("error should not be nil")
		}
		if !errors.Is(e, e) {
			t.Errorf("error.Is should match itself: %v", e)
		}
	}
}
