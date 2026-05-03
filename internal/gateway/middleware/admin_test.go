package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	mw "github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
)

func TestAdminOnly_NoUser(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := AdminOnly()(next)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
	if called {
		t.Error("next handler should not be called")
	}
}

func TestAdminOnly_NonAdmin(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := AdminOnly()(next)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := mw.SetUserInContext(req.Context(), &domain.User{ID: 1, IsAdmin: false})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
	if called {
		t.Error("next handler should not be called")
	}
}

func TestAdminOnly_Admin(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := AdminOnly()(next)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := mw.SetUserInContext(req.Context(), &domain.User{ID: 1, IsAdmin: true})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !called {
		t.Error("next handler should be called")
	}
}
