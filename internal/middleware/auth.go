package middleware

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

const userContextKey contextKey = "user"

type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionID string) (*domain.User, error)
}

func Auth(validator SessionValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			user, err := validator.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *domain.User {
	user, _ := ctx.Value(userContextKey).(*domain.User)
	return user
}
