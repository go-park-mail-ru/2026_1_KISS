package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	var resp response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "bad request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	var resp response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error != "bad request" {
		t.Errorf("expected error 'bad request', got %q", resp.Error)
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	t.Run("valid json", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"test"}`))
		var p payload
		if err := DecodeJSON(r, &p); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if p.Name != "test" {
			t.Errorf("expected name 'test', got %q", p.Name)
		}
	})

	t.Run("nil body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Body = nil
		var p payload
		if err := DecodeJSON(r, &p); err == nil {
			t.Error("expected error for nil body")
		}
	})

	t.Run("unknown fields", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"test","unknown":"field"}`))
		var p payload
		if err := DecodeJSON(r, &p); err == nil {
			t.Error("expected error for unknown fields")
		}
	})
}
