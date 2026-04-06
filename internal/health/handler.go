package health

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type pinger interface {
	PingContext(ctx context.Context) error
}

type Handler struct {
	db pinger
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.Health)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		httputil.Error(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
