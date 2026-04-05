package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/mail"
)

type mockUserRepo struct {
	createFn         func(ctx context.Context, user *domain.User) (int64, error)
	getByIDFn        func(ctx context.Context, id int64) (*domain.User, error)
	getByEmailFn     func(ctx context.Context, email string) (*domain.User, error)
	updateVerifiedFn func(ctx context.Context, userID int64) error
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
func (m *mockUserRepo) UpdateVerified(ctx context.Context, userID int64) error {
	if m.updateVerifiedFn != nil {
		return m.updateVerifiedFn(ctx, userID)
	}
	return nil
}

type mockVerificationRepo struct {
	createFn func(ctx context.Context, v *domain.VerificationToken) error
	getFn    func(ctx context.Context, token string) (*domain.VerificationToken, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (m *mockVerificationRepo) Create(ctx context.Context, v *domain.VerificationToken) error {
	if m.createFn != nil {
		return m.createFn(ctx, v)
	}
	return nil
}

func (m *mockVerificationRepo) GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error) {
	if m.getFn != nil {
		return m.getFn(ctx, token)
	}
	return nil, nil
}

func (m *mockVerificationRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
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

func newUsecase(
	userRepo *mockUserRepo,
	sessionRepo *mockSessionRepo,
	verificationRepo *mockVerificationRepo,
) *usecase.AuthUsecase {
	return usecase.New(
		userRepo,
		sessionRepo,
		verificationRepo,
		mail.New(),
		24*time.Hour,
	)
}

func TestRegister_Success(t *testing.T) {
	userRepo := &mockUserRepo{
		createFn: func(ctx context.Context, user *domain.User) (int64, error) {
			return 1, nil
		},
	}
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})

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
	uc := newUsecase(&mockUserRepo{}, &mockSessionRepo{}, &mockVerificationRepo{})
	_, err := uc.Register(context.Background(), "user", "invalid", "password123")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	userRepo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, PasswordHash: string(hash), IsVerified: true}, nil
		},
	}
	sessionRepo := &mockSessionRepo{
		createFn: func(ctx context.Context, session *domain.Session) error { return nil },
	}
	uc := newUsecase(userRepo, sessionRepo, &mockVerificationRepo{})

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
			return &domain.User{
				ID:           1,
				PasswordHash: string(hash),
				IsVerified:   true,
			}, nil
		},
	}

	sessionRepo := &mockSessionRepo{
		createFn: func(ctx context.Context, session *domain.Session) error { return nil },
	}

	uc := newUsecase(userRepo, sessionRepo, &mockVerificationRepo{})

	_, _, err := uc.Login(context.Background(), "test@example.com", "wrong")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestValidateSession_Success(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return &domain.Session{
				ID:        id,
				UserID:    1,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: 1}, nil
		},
	}

	uc := newUsecase(userRepo, sessionRepo, &mockVerificationRepo{})

	user, err := uc.ValidateSession(context.Background(), "valid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("want 1, got %d", user.ID)
	}
}

func TestValidateSession_Expired(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return &domain.Session{
				ID:        id,
				UserID:    1,
				ExpiresAt: time.Now().Add(-time.Hour),
			}, nil
		},
		deleteByIDFn: func(ctx context.Context, id string) error { return nil },
	}

	uc := newUsecase(&mockUserRepo{}, sessionRepo, &mockVerificationRepo{})

	_, err := uc.ValidateSession(context.Background(), "expired")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	uc := newUsecase(&mockUserRepo{}, &mockSessionRepo{}, &mockVerificationRepo{})
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
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})
	_, err := uc.Register(context.Background(), "testuser", "test@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("want ErrConflict, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})
	_, _, err := uc.Login(context.Background(), "no@example.com", "password123")
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
	uc := newUsecase(&mockUserRepo{}, sessionRepo, &mockVerificationRepo{})
	_, err := uc.ValidateSession(context.Background(), "missing-session")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestRegister_InvalidUsername(t *testing.T) {
	uc := newUsecase(&mockUserRepo{}, &mockSessionRepo{}, &mockVerificationRepo{})
	_, err := uc.Register(context.Background(), "a!", "test@example.com", "password123")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestLogin_SessionCreateError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	userRepo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, PasswordHash: string(hash), IsVerified: true}, nil
		},
	}
	sessionRepo := &mockSessionRepo{
		createFn: func(ctx context.Context, session *domain.Session) error {
			return errors.New("db error")
		},
	}
	uc := newUsecase(userRepo, sessionRepo, &mockVerificationRepo{})
	_, _, err := uc.Login(context.Background(), "test@example.com", "password123")
	if err == nil {
		t.Error("expected error")
	}
}

func TestLogout(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		deleteByIDFn: func(ctx context.Context, id string) error {
			return nil
		},
	}
	uc := newUsecase(&mockUserRepo{}, sessionRepo, &mockVerificationRepo{})
	err := uc.Logout(context.Background(), "some-session")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSession_UserNotFound(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return &domain.Session{
				ID:        id,
				UserID:    1,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := newUsecase(userRepo, sessionRepo, &mockVerificationRepo{})

	_, err := uc.ValidateSession(context.Background(), "valid-session")

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}
