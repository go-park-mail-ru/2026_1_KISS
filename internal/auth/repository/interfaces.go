//go:generate mockgen -destination=../../mocks/auth_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository UserRepository,SessionRepository
package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) error
	UpdateProfile(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
	UpdateEmail(ctx context.Context, userID int64, email string) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	DeleteByID(ctx context.Context, id string) error
}
