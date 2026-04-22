package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Content-Security-Policy", "default-src 'self'"},
		{"X-XSS-Protection", "0"},
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tc := range tests {
		got := rec.Header().Get(tc.header)
		if got != tc.want {
			t.Errorf("%s: want %q, got %q", tc.header, tc.want, got)
		}
	}
}
