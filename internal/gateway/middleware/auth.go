package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	mw "github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

func Auth(authClient pb.AuthServiceClient) mw.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			resp, err := authClient.ValidateSession(r.Context(), &pb.ValidateSessionRequest{
				SessionId: cookie.Value,
			})
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			info := resp.GetUser()
			user := &domain.User{
				ID:          info.GetId(),
				Username:    info.GetUsername(),
				Email:       info.GetEmail(),
				AvatarURL:   info.GetAvatarUrl(),
				Status:      info.GetStatus(),
				Description: info.GetDescription(),
				IsVerified:  info.GetIsVerified(),
				IsAdmin:     info.GetIsAdmin(),
				Plan:        info.GetPlan(),
				CreatedAt:   time.Unix(info.GetCreatedAt(), 0),
				UpdatedAt:   time.Unix(info.GetUpdatedAt(), 0),
			}
			ctx := mw.SetUserInContext(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *domain.User {
	return mw.UserFromContext(ctx)
}
