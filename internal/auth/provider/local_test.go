package provider

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
)

func TestLocalProvider_Name(t *testing.T) {
	p := NewLocalProvider(nil)
	if p.Name() != "local" {
		t.Errorf("want local, got %s", p.Name())
	}
}

func TestLocalProvider_Authenticate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepository(ctrl)

	hash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(&domain.User{
		ID:           1,
		Email:        "test@example.com",
		PasswordHash: string(hash),
	}, nil)

	p := NewLocalProvider(userRepo)
	user, err := p.Authenticate(context.Background(), map[string]string{
		"email":    "test@example.com",
		"password": "Password123!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("want user id 1, got %d", user.ID)
	}
}

func TestLocalProvider_Authenticate_WrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepository(ctrl)

	hash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(&domain.User{
		ID:           1,
		PasswordHash: string(hash),
	}, nil)

	p := NewLocalProvider(userRepo)
	_, err := p.Authenticate(context.Background(), map[string]string{
		"email":    "test@example.com",
		"password": "wrong",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestLocalProvider_Authenticate_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepository(ctrl)

	userRepo.EXPECT().GetByEmail(gomock.Any(), "no@example.com").Return(nil, domain.ErrNotFound)

	p := NewLocalProvider(userRepo)
	_, err := p.Authenticate(context.Background(), map[string]string{
		"email":    "no@example.com",
		"password": "Password123!",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestLocalProvider_Authenticate_MissingCredentials(t *testing.T) {
	p := NewLocalProvider(nil)
	_, err := p.Authenticate(context.Background(), map[string]string{})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}
