package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic recovered: %v\n%s", rec, debug.Stack())
					httputil.Error(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
