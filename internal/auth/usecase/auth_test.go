package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
)

func TestRegister_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	userRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(int64(1), nil)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	user, err := uc.Register(context.Background(), "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("want ID=1, got %d", user.ID)
	}
	if user.PasswordHash == "password123" {
		t.Error("password should be hashed")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.Register(context.Background(), "user", "invalid", "password123")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.Register(context.Background(), "user", "test@example.com", "short")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestRegister_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	userRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(int64(0), domain.ErrConflict)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.Register(context.Background(), "testuser", "test@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("want ErrConflict, got %v", err)
	}
}

func TestRegister_InvalidUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.Register(context.Background(), "a!", "test@example.com", "password123")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	userRepo.EXPECT().
		GetByEmail(gomock.Any(), "test@example.com").
		Return(&domain.User{ID: 1, Email: "test@example.com", PasswordHash: string(hash)}, nil)

	sessionRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	session, user, err := uc.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil || session.ID == "" {
		t.Error("expected valid session")
	}
	if user == nil || user.ID != 1 {
		t.Error("expected valid user")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	userRepo.EXPECT().
		GetByEmail(gomock.Any(), "test@example.com").
		Return(&domain.User{ID: 1, PasswordHash: string(hash)}, nil)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, _, err := uc.Login(context.Background(), "test@example.com", "wrong")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	userRepo.EXPECT().
		GetByEmail(gomock.Any(), "no@example.com").
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, _, err := uc.Login(context.Background(), "no@example.com", "password123")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestLogin_SessionCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	userRepo.EXPECT().
		GetByEmail(gomock.Any(), "test@example.com").
		Return(&domain.User{ID: 1, Email: "test@example.com", PasswordHash: string(hash)}, nil)

	sessionRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.New("db error"))

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, _, err := uc.Login(context.Background(), "test@example.com", "password123")
	if err == nil {
		t.Error("expected error")
	}
}

func TestLogout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	sessionRepo.EXPECT().
		DeleteByID(gomock.Any(), "some-session").
		Return(nil)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	err := uc.Logout(context.Background(), "some-session")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSession_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	sessionRepo.EXPECT().
		GetByID(gomock.Any(), "valid-session").
		Return(&domain.Session{ID: "valid-session", UserID: 1, ExpiresAt: time.Now().Add(time.Hour)}, nil)

	userRepo.EXPECT().
		GetByID(gomock.Any(), int64(1)).
		Return(&domain.User{ID: 1}, nil)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	got, err := uc.ValidateSession(context.Background(), "valid-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("want user ID=1, got %d", got.ID)
	}
}

func TestValidateSession_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	sessionRepo.EXPECT().
		GetByID(gomock.Any(), "expired-session").
		Return(&domain.Session{ID: "expired-session", UserID: 1, ExpiresAt: time.Now().Add(-time.Hour)}, nil)

	sessionRepo.EXPECT().
		DeleteByID(gomock.Any(), "expired-session").
		Return(nil)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.ValidateSession(context.Background(), "expired-session")
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}

func TestValidateSession_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	sessionRepo.EXPECT().
		GetByID(gomock.Any(), "missing-session").
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.ValidateSession(context.Background(), "missing-session")
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}

func TestValidateSession_ExpiredFromRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	sessionRepo.EXPECT().
		GetByID(gomock.Any(), "expired-session").
		Return(nil, domain.ErrSessionExpired)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.ValidateSession(context.Background(), "expired-session")
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}

func TestValidateSession_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)

	sessionRepo.EXPECT().
		GetByID(gomock.Any(), "valid-session").
		Return(&domain.Session{ID: "valid-session", UserID: 1, ExpiresAt: time.Now().Add(time.Hour)}, nil)

	userRepo.EXPECT().
		GetByID(gomock.Any(), int64(1)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(userRepo, sessionRepo, 24*time.Hour)

	_, err := uc.ValidateSession(context.Background(), "valid-session")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}
