package middleware

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/handlers"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/models"
)

type contextKey string

const UserContextKey contextKey = "user"

// Проверка авторизации пользователя
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			handlers.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		//TODO: Проверка токен
		token := cookie.Value
		user, err := validateToken(token)
		if err != nil {
			handlers.RespondError(w, http.StatusUnauthorized, "Invalid session")
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// Проверка токена и возврат пользователя
func validateToken(token string) (*models.User, error) {
	// TODO: Проверка JWT/ сессии в БД
	if token == "test-session" {
		return &models.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}, nil
	}
	return nil, http.ErrNoCookie
}

// Получение пользователя из контекста
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
