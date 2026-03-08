package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type AuthUsecase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	sessionTTL  time.Duration
}

func New(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, sessionTTL time.Duration) *AuthUsecase {
	return &AuthUsecase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		sessionTTL:  sessionTTL,
	}
}

func (uc *AuthUsecase) Register(ctx context.Context, username, email, password string) (*domain.User, error) {
	if err := httputil.ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}
	if err := httputil.ValidatePassword(password); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}
	if err := httputil.ValidateUsername(username); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
	}

	id, err := uc.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	user.ID = id
	return user, nil
}

func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (*domain.Session, *domain.User, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, domain.ErrUnauthorized
	}

	session := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(uc.sessionTTL),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return session, user, nil
}

func (uc *AuthUsecase) Logout(ctx context.Context, sessionID string) error {
	_ = uc.sessionRepo.DeleteByID(ctx, sessionID)
	return nil
}

func (uc *AuthUsecase) ValidateSession(ctx context.Context, sessionID string) (*domain.User, error) {
	session, err := uc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if session.IsExpired() {
		_ = uc.sessionRepo.DeleteByID(ctx, sessionID)
		return nil, domain.ErrUnauthorized
	}

	user, err := uc.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	return user, nil
}
