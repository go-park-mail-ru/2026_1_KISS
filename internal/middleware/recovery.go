package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error(r.Context(), "panic recovered",
						"error", rec,
						"stack", string(debug.Stack()),
					)
					httputil.Error(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
