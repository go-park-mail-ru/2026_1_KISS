package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/provider"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
)

type stubProvider struct {
	name string
	info *domain.ExternalUserInfo
	err  error
}

func (p *stubProvider) Name() string { return p.name }
func (p *stubProvider) AuthorizationURL(state, challenge string) string {
	return "https://provider.example/auth?state=" + state + "&c=" + challenge
}
func (p *stubProvider) Exchange(_ context.Context, _, _ string) (*domain.ExternalUserInfo, error) {
	return p.info, p.err
}

type oauthDeps struct {
	userRepo  *mocks.MockUserRepository
	sessRepo  *mocks.MockSessionRepository
	oauthRepo *mocks.MockOAuthAccountRepository
	stateRepo *mocks.MockOAuthStateRepository
	stub      *stubProvider
	uc        *usecase.OAuthUsecase
}

func newOAuthEnv(t *testing.T, info *domain.ExternalUserInfo) *oauthDeps {
	t.Helper()
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepository(ctrl)
	sessRepo := mocks.NewMockSessionRepository(ctrl)
	oauthRepo := mocks.NewMockOAuthAccountRepository(ctrl)
	stateRepo := mocks.NewMockOAuthStateRepository(ctrl)
	stub := &stubProvider{name: domain.OAuthProviderGoogle, info: info}
	registry := provider.Registry{stub.name: stub}
	uc := usecase.NewOAuthUsecase(userRepo, sessRepo, oauthRepo, stateRepo, registry, time.Hour, 5*time.Minute)
	return &oauthDeps{userRepo, sessRepo, oauthRepo, stateRepo, stub, uc}
}

func TestOAuthUsecase_Start_HappyPath(t *testing.T) {
	d := newOAuthEnv(t, nil)
	d.stateRepo.EXPECT().Save(gomock.Any(), gomock.Any(), 5*time.Minute).Return(nil)

	authURL, state, expAt, err := d.uc.Start(context.Background(), domain.OAuthProviderGoogle)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if authURL == "" || state == "" || expAt.IsZero() {
		t.Fatalf("expected non-empty url+state, got %q %q %v", authURL, state, expAt)
	}
}

func TestOAuthUsecase_Start_UnknownProvider(t *testing.T) {
	d := newOAuthEnv(t, nil)
	_, _, _, err := d.uc.Start(context.Background(), "unknown")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestOAuthUsecase_Callback_ExistingOAuthAccount(t *testing.T) {
	info := &domain.ExternalUserInfo{ProviderID: "g-1", Email: "u@e.com", EmailVerified: true, Username: "Ivan"}
	d := newOAuthEnv(t, info)

	d.stateRepo.EXPECT().Consume(gomock.Any(), "st").Return(&domain.OAuthState{State: "st", Provider: domain.OAuthProviderGoogle, CodeVerifier: "v"}, nil)
	d.oauthRepo.EXPECT().GetByProviderID(gomock.Any(), domain.OAuthProviderGoogle, "g-1").
		Return(&domain.OAuthAccount{ID: 1, UserID: 42, Provider: domain.OAuthProviderGoogle, ProviderID: "g-1"}, nil)
	d.userRepo.EXPECT().GetByID(gomock.Any(), int64(42)).Return(&domain.User{ID: 42, Email: "u@e.com", IsVerified: true}, nil)
	d.sessRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	session, user, err := d.uc.Callback(context.Background(), domain.OAuthProviderGoogle, "code", "st")
	if err != nil {
		t.Fatalf("Callback: %v", err)
	}
	if session.UserID != 42 || user.ID != 42 {
		t.Fatalf("unexpected: %+v %+v", session, user)
	}
}

func TestOAuthUsecase_Callback_LinkByVerifiedEmail(t *testing.T) {
	info := &domain.ExternalUserInfo{ProviderID: "g-2", Email: "u@e.com", EmailVerified: true, Username: "Ivan"}
	d := newOAuthEnv(t, info)

	d.stateRepo.EXPECT().Consume(gomock.Any(), "st").Return(&domain.OAuthState{Provider: domain.OAuthProviderGoogle, CodeVerifier: "v"}, nil)
	d.oauthRepo.EXPECT().GetByProviderID(gomock.Any(), domain.OAuthProviderGoogle, "g-2").Return(nil, domain.ErrNotFound)
	d.userRepo.EXPECT().GetByEmail(gomock.Any(), "u@e.com").Return(&domain.User{ID: 7, Email: "u@e.com", IsVerified: true}, nil)
	d.oauthRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(int64(11), nil)
	d.sessRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	_, user, err := d.uc.Callback(context.Background(), domain.OAuthProviderGoogle, "code", "st")
	if err != nil {
		t.Fatalf("Callback: %v", err)
	}
	if user.ID != 7 {
		t.Fatalf("expected linked existing user 7, got %d", user.ID)
	}
}

func TestOAuthUsecase_Callback_RejectUnverifiedExistingEmail(t *testing.T) {
	info := &domain.ExternalUserInfo{ProviderID: "g-3", Email: "u@e.com", EmailVerified: true, Username: "X"}
	d := newOAuthEnv(t, info)

	d.stateRepo.EXPECT().Consume(gomock.Any(), "st").Return(&domain.OAuthState{Provider: domain.OAuthProviderGoogle, CodeVerifier: "v"}, nil)
	d.oauthRepo.EXPECT().GetByProviderID(gomock.Any(), domain.OAuthProviderGoogle, "g-3").Return(nil, domain.ErrNotFound)
	d.userRepo.EXPECT().GetByEmail(gomock.Any(), "u@e.com").Return(&domain.User{ID: 5, Email: "u@e.com", IsVerified: false}, nil)

	_, _, err := d.uc.Callback(context.Background(), domain.OAuthProviderGoogle, "code", "st")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestOAuthUsecase_Callback_NewUserCreated(t *testing.T) {
	info := &domain.ExternalUserInfo{ProviderID: "g-9", Email: "new@e.com", EmailVerified: true, Username: "Petya"}
	d := newOAuthEnv(t, info)

	d.stateRepo.EXPECT().Consume(gomock.Any(), "st").Return(&domain.OAuthState{Provider: domain.OAuthProviderGoogle, CodeVerifier: "v"}, nil)
	d.oauthRepo.EXPECT().GetByProviderID(gomock.Any(), domain.OAuthProviderGoogle, "g-9").Return(nil, domain.ErrNotFound)
	d.userRepo.EXPECT().GetByEmail(gomock.Any(), "new@e.com").Return(nil, domain.ErrNotFound)
	d.userRepo.EXPECT().GetByUsername(gomock.Any(), "Petya").Return(nil, domain.ErrNotFound)
	d.userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(int64(101), nil)
	d.userRepo.EXPECT().SetVerified(gomock.Any(), int64(101), true).Return(nil)
	d.oauthRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(int64(50), nil)
	d.sessRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	_, user, err := d.uc.Callback(context.Background(), domain.OAuthProviderGoogle, "code", "st")
	if err != nil {
		t.Fatalf("Callback: %v", err)
	}
	if user.ID != 101 || user.Username != "Petya" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestOAuthUsecase_Callback_StateExpired(t *testing.T) {
	d := newOAuthEnv(t, nil)
	d.stateRepo.EXPECT().Consume(gomock.Any(), "bad").Return(nil, domain.ErrNotFound)

	_, _, err := d.uc.Callback(context.Background(), domain.OAuthProviderGoogle, "c", "bad")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}
