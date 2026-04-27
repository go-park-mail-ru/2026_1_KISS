package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/mail"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/sanitize"
)

type AuthUsecase struct {
	userRepo         repository.UserRepository
	sessionRepo      repository.SessionRepository
	sessionTTL       time.Duration
	verificationRepo repository.VerificationRepository
	mailService      mail.Sender
}

// func New(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, sessionTTL time.Duration) *AuthUsecase {
// 	return &AuthUsecase{
// 		userRepo:    userRepo,
// 		sessionRepo: sessionRepo,
// 		sessionTTL:  sessionTTL,
// 	}
// }

func New(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	verificationRepo repository.VerificationRepository,
	mailService mail.Sender,
	sessionTTL time.Duration,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		verificationRepo: verificationRepo,
		mailService:      mailService,
		sessionTTL:       sessionTTL,
	}
}

func (uc *AuthUsecase) Register(ctx context.Context, username, email, password string) (*domain.User, error) {
	logger.Info(ctx, "REGISTER START",
		"email", email,
	)

	// 1. validation email
	if err := httputil.ValidateEmail(email); err != nil {
		logger.Error(ctx, "REGISTER FAILED",
			"step", "validate_email",
			"error", err,
		)
		return nil, fmt.Errorf("%w: invalid email format", domain.ErrInvalidInput)
	}
	logger.Info(ctx, "REGISTER STEP OK", "step", "validate_email")

	// 2. password
	if err := httputil.ValidatePassword(password); err != nil {
		logger.Error(ctx, "REGISTER FAILED",
			"step", "validate_password",
			"error", err,
		)
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}
	logger.Info(ctx, "REGISTER STEP OK", "step", "validate_password")

	// 3. username
	if err := httputil.ValidateUsername(username); err != nil {
		logger.Error(ctx, "REGISTER FAILED",
			"step", "validate_username",
			"error", err,
		)
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}
	logger.Info(ctx, "REGISTER STEP OK", "step", "validate_username")

	// 4. hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error(ctx, "REGISTER FAILED",
			"step", "hash_password",
			"error", err,
		)
		return nil, fmt.Errorf("hash password: %w", err)
	}
	logger.Info(ctx, "REGISTER STEP OK", "step", "hash_password")

	// 5. create user
	user := &domain.User{
		Username:     sanitize.EscapeHTML(username),
		Email:        email,
		PasswordHash: string(hash),
		IsVerified:   false,
	}

	id, err := uc.userRepo.Create(ctx, user)
	if err != nil {
		logger.Error(ctx, "REGISTER FAILED",
			"step", "user_create",
			"error", err,
		)
		return nil, err
	}

	logger.Info(ctx, "REGISTER STEP OK",
		"step", "user_create",
		"user_id", id,
	)

	user.ID = id

	// 6. verification token
	token := uuid.New().String()

	vt := &domain.VerificationToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := uc.verificationRepo.Create(ctx, vt); err != nil {
		logger.Error(ctx, "REGISTER FAILED",
			"step", "verification_create",
			"error", err,
		)
		return nil, err
	}

	logger.Info(ctx, "REGISTER STEP OK",
		"step", "verification_create",
	)

	// 7. send email START
	logger.Info(ctx, "EMAIL SENDING START",
		"email", user.Email,
		"user_id", user.ID,
	)

	if err := uc.mailService.SendVerification(user.Email, token); err != nil {
		logger.Error(ctx, "EMAIL SEND FAILED",
			"step", "send_verification",
			"email", user.Email,
			"user_id", user.ID,
			"error", err,
		)
		return nil, fmt.Errorf("send verification email: %w", err)
	}

	logger.Info(ctx, "EMAIL SENT OK",
		"user_id", user.ID,
	)

	logger.Info(ctx, "REGISTER SUCCESS",
		"user_id", user.ID,
	)

	return user, nil
}

func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (*domain.Session, *domain.User, error) {
	logger.Info(ctx, "usecase.auth.Login", "email", email)

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		logger.Error(ctx, "usecase.auth.Login", "error", domain.ErrUnauthorized)
		return nil, nil, domain.ErrUnauthorized
	}

	if !user.IsVerified {
		return nil, nil, domain.ErrForbidden
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		logger.Error(ctx, "usecase.auth.Login", "error", domain.ErrUnauthorized)
		return nil, nil, domain.ErrUnauthorized
	}

	session := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(uc.sessionTTL),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		logger.Error(ctx, "usecase.auth.Login", "error", err)
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	logger.Info(ctx, "usecase.auth.Login", "user_id", user.ID)
	return session, user, nil
}

func (uc *AuthUsecase) Logout(ctx context.Context, sessionID string) error {
	logger.Info(ctx, "usecase.auth.Logout", "session_id", sessionID)
	if err := uc.sessionRepo.DeleteByID(ctx, sessionID); err != nil {
		logger.Error(ctx, "usecase.auth.Logout", "error", err, "session_id", sessionID)
		return fmt.Errorf("delete session: %w", err)
	}
	logger.Info(ctx, "usecase.auth.Logout", "status", "ok")
	return nil
}

func (uc *AuthUsecase) ValidateSession(ctx context.Context, sessionID string) (*domain.User, error) {
	logger.Info(ctx, "usecase.auth.ValidateSession", "session_id", sessionID)

	session, err := uc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		logger.Error(ctx, "usecase.auth.ValidateSession", "error", err)
		switch {
		case errors.Is(err, domain.ErrSessionExpired), errors.Is(err, domain.ErrNotFound):
			return nil, domain.ErrSessionExpired
		default:
			return nil, domain.ErrUnauthorized
		}
	}

	if session.IsExpired() {
		logger.Error(ctx, "usecase.auth.ValidateSession", "error", "session expired")
		_ = uc.sessionRepo.DeleteByID(ctx, sessionID)
		return nil, domain.ErrSessionExpired
	}

	user, err := uc.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		logger.Error(ctx, "usecase.auth.ValidateSession", "error", err)
		return nil, domain.ErrUnauthorized
	}

	logger.Info(ctx, "usecase.auth.ValidateSession", "user_id", user.ID)
	return user, nil
}

func (uc *AuthUsecase) ConfirmEmail(ctx context.Context, token string) error {
	vt, err := uc.verificationRepo.GetByToken(ctx, token)
	if err != nil {
		return err
	}

	if time.Now().After(vt.ExpiresAt) {
		return domain.ErrInvalidInput
	}

	if err := uc.userRepo.SetVerified(ctx, vt.UserID, true); err != nil {
		return err
	}

	return uc.verificationRepo.Delete(ctx, vt.ID)
}
func (uc *AuthUsecase) GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	logger.Info(ctx, "usecase.auth.GetUserByIdentifier", "identifier", identifier)
	if strings.Contains(identifier, "@") {
		return uc.userRepo.GetByEmail(ctx, identifier)
	}
	return uc.userRepo.GetByUsername(ctx, identifier)
}
