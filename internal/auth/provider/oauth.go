package provider

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type OAuthProvider interface {
	Name() string
	AuthorizationURL(state, codeChallenge string) string
	Exchange(ctx context.Context, code, codeVerifier string) (*domain.ExternalUserInfo, error)
}
