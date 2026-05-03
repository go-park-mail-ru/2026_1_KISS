package provider

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type AuthProvider interface {
	Name() string
	Authenticate(ctx context.Context, credentials map[string]string) (*domain.User, error)
}
