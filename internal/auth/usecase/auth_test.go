package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type mockUserRepo struct {
	createFn     func(ctx context.Context, user *domain.User) (int64, error)
	getByIDFn    func(ctx context.Context, id int64) (*domain.User, error)
	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (int64, error) {
	return m.createFn(ctx, user)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.getByEmailFn(ctx, email)
}

type mockSessionRepo struct {
	createFn     func(ctx context.Context, session *domain.Session) error
	getByIDFn    func(ctx context.Context, id string) (*domain.Session, error)
	deleteByIDFn func(ctx context.Context, id string) error
}

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.Session) error {
	return m.createFn(ctx, session)
}

func (m *mockSessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockSessionRepo) DeleteByID(ctx context.Context, id string) error {
	return m.deleteByIDFn(ctx, id)
}

func TestRegister_Success(t *testing.T) {
	userRepo := &mockUserRepo{
		createFn: func(ctx context.Context, user *domain.User) (int64, error) {
			return 1, nil
		},
	}
	sessionRepo := &mockSessionRepo{}
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
	uc := usecase.New(&mockUserRepo{}, &mockSessionRepo{}, 24*time.Hour)
	_, err := uc.Register(context.Background(), "user", "invalid", "password123")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	uc := usecase.New(&mockUserRepo{}, &mockSessionRepo{}, 24*time.Hour)
	_, err := uc.Register(context.Background(), "user", "test@example.com", "short")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestRegister_Conflict(t *testing.T) {
	userRepo := &mockUserRepo{
		createFn: func(ctx context.Context, user *domain.User) (int64, error) {
			return 0, domain.ErrConflict
		},
	}
	uc := usecase.New(userRepo, &mockSessionRepo{}, 24*time.Hour)
	_, err := uc.Register(context.Background(), "testuser", "test@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("want ErrConflict, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	userRepo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, PasswordHash: string(hash)}, nil
		},
	}
	sessionRepo := &mockSessionRepo{
		createFn: func(ctx context.Context, session *domain.Session) error {
			return nil
		},
	}
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
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	userRepo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, PasswordHash: string(hash)}, nil
		},
	}
	uc := usecase.New(userRepo, &mockSessionRepo{}, 24*time.Hour)
	_, _, err := uc.Login(context.Background(), "test@example.com", "wrong")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(userRepo, &mockSessionRepo{}, 24*time.Hour)
	_, _, err := uc.Login(context.Background(), "no@example.com", "password123")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestValidateSession_Success(t *testing.T) {
	user := &domain.User{ID: 1}
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return &domain.Session{ID: id, UserID: 1, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.User, error) {
			return user, nil
		},
	}
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
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return &domain.Session{ID: id, UserID: 1, ExpiresAt: time.Now().Add(-time.Hour)}, nil
		},
		deleteByIDFn: func(ctx context.Context, id string) error {
			return nil
		},
	}
	uc := usecase.New(&mockUserRepo{}, sessionRepo, 24*time.Hour)
	_, err := uc.ValidateSession(context.Background(), "expired-session")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestValidateSession_NotFound(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(&mockUserRepo{}, sessionRepo, 24*time.Hour)
	_, err := uc.ValidateSession(context.Background(), "missing-session")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}
