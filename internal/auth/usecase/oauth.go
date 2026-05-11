package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/provider"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

const (
	usernameCollisionAttempts = 10
	oauthDefaultUsername      = "user"
)

type OAuthUsecase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	oauthRepo   repository.OAuthAccountRepository
	stateRepo   repository.OAuthStateRepository
	providers   provider.Registry
	sessionTTL  time.Duration
	stateTTL    time.Duration
}

func NewOAuthUsecase(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	oauthRepo repository.OAuthAccountRepository,
	stateRepo repository.OAuthStateRepository,
	providers provider.Registry,
	sessionTTL, stateTTL time.Duration,
) *OAuthUsecase {
	return &OAuthUsecase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		oauthRepo:   oauthRepo,
		stateRepo:   stateRepo,
		providers:   providers,
		sessionTTL:  sessionTTL,
		stateTTL:    stateTTL,
	}
}

func (uc *OAuthUsecase) Start(ctx context.Context, providerName string) (string, string, time.Time, error) {
	p, err := uc.providers.Get(providerName)
	if err != nil {
		return "", "", time.Time{}, err
	}

	state, err := provider.GenerateState()
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate state: %w", err)
	}
	verifier, err := provider.GenerateCodeVerifier()
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate code_verifier: %w", err)
	}
	challenge := provider.CodeChallengeS256(verifier)

	st := &domain.OAuthState{
		State:        state,
		Provider:     p.Name(),
		CodeVerifier: verifier,
	}
	if err := uc.stateRepo.Save(ctx, st, uc.stateTTL); err != nil {
		return "", "", time.Time{}, fmt.Errorf("save state: %w", err)
	}

	expiresAt := time.Now().Add(uc.stateTTL)
	authURL := p.AuthorizationURL(state, challenge)
	logger.Info(ctx, "usecase.oauth.Start", "provider", p.Name())
	return authURL, state, expiresAt, nil
}

func (uc *OAuthUsecase) Callback(ctx context.Context, providerName, code, state string) (*domain.Session, *domain.User, error) {
	if code == "" || state == "" {
		return nil, nil, fmt.Errorf("%w: code and state are required", domain.ErrInvalidInput)
	}

	p, err := uc.providers.Get(providerName)
	if err != nil {
		return nil, nil, err
	}

	stState, err := uc.stateRepo.Consume(ctx, state)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: state expired or unknown", domain.ErrUnauthorized)
		}
		return nil, nil, fmt.Errorf("consume state: %w", err)
	}
	if stState.Provider != providerName {
		return nil, nil, fmt.Errorf("%w: state/provider mismatch", domain.ErrUnauthorized)
	}

	info, err := p.Exchange(ctx, code, stState.CodeVerifier)
	if err != nil {
		logger.Error(ctx, "usecase.oauth.Callback", "step", "exchange", "error", err, "provider", providerName)
		return nil, nil, fmt.Errorf("%w: oauth exchange failed", domain.ErrUnauthorized)
	}

	user, err := uc.resolveUser(ctx, providerName, info)
	if err != nil {
		return nil, nil, err
	}

	session, err := uc.createSession(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	logger.Info(ctx, "usecase.oauth.Callback", "provider", providerName, "user_id", user.ID)
	return session, user, nil
}

func (uc *OAuthUsecase) resolveUser(ctx context.Context, providerName string, info *domain.ExternalUserInfo) (*domain.User, error) {
	acc, err := uc.oauthRepo.GetByProviderID(ctx, providerName, info.ProviderID)
	if err == nil {
		user, getErr := uc.userRepo.GetByID(ctx, acc.UserID)
		if getErr != nil {
			return nil, fmt.Errorf("load oauth user: %w", getErr)
		}
		return user, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("lookup oauth account: %w", err)
	}

	if info.Email != "" && info.EmailVerified {
		existing, getErr := uc.userRepo.GetByEmail(ctx, info.Email)
		if getErr == nil {
			if !existing.IsVerified {
				return nil, fmt.Errorf("%w: email belongs to unverified local account", domain.ErrConflict)
			}
			if _, linkErr := uc.oauthRepo.Create(ctx, &domain.OAuthAccount{
				UserID:     existing.ID,
				Provider:   providerName,
				ProviderID: info.ProviderID,
			}); linkErr != nil && !errors.Is(linkErr, domain.ErrConflict) {
				return nil, fmt.Errorf("link oauth account: %w", linkErr)
			}
			return existing, nil
		} else if !errors.Is(getErr, domain.ErrNotFound) {
			return nil, fmt.Errorf("lookup user by email: %w", getErr)
		}
	}

	return uc.createOAuthUser(ctx, providerName, info)
}

func (uc *OAuthUsecase) createOAuthUser(ctx context.Context, providerName string, info *domain.ExternalUserInfo) (*domain.User, error) {
	username, err := uc.generateUsername(ctx, info.Username)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Username:     username,
		Email:        info.Email,
		PasswordHash: "",
		IsVerified:   info.EmailVerified || info.Email == "",
		AvatarURL:    info.AvatarURL,
	}

	id, err := uc.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("create oauth user: %w", err)
	}
	user.ID = id

	if user.IsVerified {
		if err := uc.userRepo.SetVerified(ctx, id, true); err != nil {
			logger.Error(ctx, "usecase.oauth.createOAuthUser", "step", "set_verified", "error", err, "user_id", id)
		}
	}
	if info.AvatarURL != "" {
		if err := uc.userRepo.UpdateAvatarURL(ctx, id, info.AvatarURL); err != nil {
			logger.Error(ctx, "usecase.oauth.createOAuthUser", "step", "set_avatar", "error", err, "user_id", id)
		}
	}

	if _, err := uc.oauthRepo.Create(ctx, &domain.OAuthAccount{
		UserID:     id,
		Provider:   providerName,
		ProviderID: info.ProviderID,
	}); err != nil {
		return nil, fmt.Errorf("attach oauth account: %w", err)
	}

	return user, nil
}

func (uc *OAuthUsecase) generateUsername(ctx context.Context, base string) (string, error) {
	cleaned := sanitizeUsername(base)
	if cleaned == "" {
		cleaned = oauthDefaultUsername
	}

	candidate := cleaned
	for i := 1; i <= usernameCollisionAttempts; i++ {
		if i > 1 {
			candidate = fmt.Sprintf("%s%d", cleaned, i)
		}
		_, err := uc.userRepo.GetByUsername(ctx, candidate)
		if errors.Is(err, domain.ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("lookup username: %w", err)
		}
	}
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:6]
	return fmt.Sprintf("%s_%s", cleaned, suffix), nil
}

func sanitizeUsername(in string) string {
	in = strings.TrimSpace(in)
	var b strings.Builder
	for _, r := range in {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}

func (uc *OAuthUsecase) createSession(ctx context.Context, userID int64) (*domain.Session, error) {
	session := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(uc.sessionTTL),
	}
	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}
