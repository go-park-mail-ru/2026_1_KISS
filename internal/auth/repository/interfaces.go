//go:generate mockgen -destination=../../mocks/auth_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository UserRepository,SessionRepository,VerificationRepository,EventRepository

package repository

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	SetVerified(ctx context.Context, userID int64, isVerified bool) error
	UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) error
	UpdateProfile(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
	UpdateEmail(ctx context.Context, userID int64, email string) error
	ListAll(ctx context.Context, limit, offset int, search string, verified *bool) ([]domain.User, int, error)
	SetBanned(ctx context.Context, userID int64, banned bool) error
	CountAll(ctx context.Context) (int64, error)
	UpdatePlan(ctx context.Context, userID int64, plan string) error
	UpdateLastActive(ctx context.Context, userID int64, t time.Time) error
	IncrementTotalTime(ctx context.Context, userID int64, seconds int64) error
	AdminUpdateUser(ctx context.Context, userID int64, username, email string) error
	DeleteUnverifiedBefore(ctx context.Context, before time.Time) (int64, error)
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

type EventRepository interface {
	Create(ctx context.Context, event *domain.UserEvent) error
	CountActiveUsers(ctx context.Context, since time.Time) (int64, error)
	CountActiveUsersByDay(ctx context.Context, since time.Time) ([]domain.DayCount, error)
	CountActiveUsersByMonth(ctx context.Context, since time.Time) ([]domain.MonthCount, error)
	CountUserActivityByDay(ctx context.Context, userID int64, since time.Time) ([]domain.DayCount, error)
}
