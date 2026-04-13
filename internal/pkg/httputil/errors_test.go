package httputil_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

func TestMapDomainError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"conflict", domain.ErrConflict, http.StatusConflict},
		{"session expired", domain.ErrSessionExpired, http.StatusUnauthorized},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized},
		{"invalid input", domain.ErrInvalidInput, http.StatusBadRequest},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden},
		{"wrapped not found", fmt.Errorf("wrap: %w", domain.ErrNotFound), http.StatusNotFound},
		{"wrapped invalid input", fmt.Errorf("wrap: %w", domain.ErrInvalidInput), http.StatusBadRequest},
		{"unknown error", errors.New("something"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			httputil.MapDomainError(rec, tc.err)
			if rec.Code != tc.wantStatus {
				t.Errorf("want status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}
