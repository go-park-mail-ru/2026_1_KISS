package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateVerified(ctx context.Context, userID int64) error
}

type VerificationRepository interface {
	Create(ctx context.Context, vt *domain.VerificationToken) error
	GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error)
	Delete(ctx context.Context, id int64) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	DeleteByID(ctx context.Context, id string) error
}
