package handler

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.Health)
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
