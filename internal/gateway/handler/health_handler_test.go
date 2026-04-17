package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Health(t *testing.T) {
	h := NewHealthHandler()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}
