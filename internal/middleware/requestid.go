package middleware

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/ctxutil"
	"github.com/google/uuid"
)

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := uuid.New().String()
			w.Header().Set("X-Request-ID", id)
			ctx := ctxutil.SetRequestID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	return ctxutil.RequestIDFromContext(ctx)
}
