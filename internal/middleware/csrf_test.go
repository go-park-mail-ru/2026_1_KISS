package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
)

func TestCSRF_SkipGET(t *testing.T) {
	handler := middleware.CSRF(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestCSRF_SkipPath(t *testing.T) {
	skip := map[string]bool{"/api/v1/auth/login": true}
	handler := middleware.CSRF(skip)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestCSRF_MissingCookie(t *testing.T) {
	handler := middleware.CSRF(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("POST", "/api/v1/notebooks", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestCSRF_MismatchToken(t *testing.T) {
	handler := middleware.CSRF(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("POST", "/api/v1/notebooks", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-a"})
	req.Header.Set("X-CSRF-Token", "token-b")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestCSRF_ValidToken(t *testing.T) {
	handler := middleware.CSRF(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("POST", "/api/v1/notebooks", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-valid"})
	req.Header.Set("X-CSRF-Token", "token-valid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestCSRF_EmptyHeader(t *testing.T) {
	handler := middleware.CSRF(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("PUT", "/api/v1/users/me", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestSetCSRFCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	token := middleware.SetCSRFCookie(rec, nil)
	if token == "" {
		t.Error("expected non-empty token")
	}
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "csrf_token" && c.Value == token {
			found = true
		}
	}
	if !found {
		t.Error("csrf_token cookie not set")
	}
}
