package provider

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type LocalProvider struct {
	userRepo repository.UserRepository
}

func NewLocalProvider(userRepo repository.UserRepository) *LocalProvider {
	return &LocalProvider{userRepo: userRepo}
}

func (p *LocalProvider) Name() string {
	return "local"
}

func (p *LocalProvider) Authenticate(ctx context.Context, credentials map[string]string) (*domain.User, error) {
	email := credentials["email"]
	password := credentials["password"]

	if email == "" || password == "" {
		return nil, fmt.Errorf("%w: email and password are required", domain.ErrInvalidInput)
	}

	user, err := p.userRepo.GetByEmail(ctx, email)
	if err != nil {
		logger.Error(ctx, "provider.local.Authenticate", "error", domain.ErrUnauthorized)
		return nil, domain.ErrUnauthorized
	}

	if user.PasswordHash == "" {
		logger.Error(ctx, "provider.local.Authenticate", "error", "oauth-only user attempted password login")
		return nil, domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		logger.Error(ctx, "provider.local.Authenticate", "error", domain.ErrUnauthorized)
		return nil, domain.ErrUnauthorized
	}

	return user, nil
}
