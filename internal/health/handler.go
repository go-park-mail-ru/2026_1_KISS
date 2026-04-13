//go:generate mockgen -source=handler.go -destination=../mocks/health_mock.go -package=mocks
package health

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

type Handler struct {
	db Pinger
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func NewWithPinger(p Pinger) *Handler {
	return &Handler{db: p}
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
