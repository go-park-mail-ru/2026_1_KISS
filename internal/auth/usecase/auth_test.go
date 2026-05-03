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
	createFn                 func(ctx context.Context, user *domain.User) (int64, error)
	getByIDFn                func(ctx context.Context, id int64) (*domain.User, error)
	getByEmailFn             func(ctx context.Context, email string) (*domain.User, error)
	getByUsernameFn          func(ctx context.Context, username string) (*domain.User, error)
	setVerifiedFn            func(ctx context.Context, userID int64, isVerified bool) error
	deleteUnverifiedBeforeFn func(ctx context.Context, before time.Time) (int64, error)
}

func (m *mockUserRepo) CountAll(_ context.Context) (int64, error) {
	return 0, nil
}
func (m *mockUserRepo) UpdatePlan(_ context.Context, _ int64, _ string) error {
	return nil
}
func (m *mockUserRepo) UpdateLastActive(_ context.Context, _ int64, _ time.Time) error {
	return nil
}
func (m *mockUserRepo) IncrementTotalTime(_ context.Context, _ int64, _ int64) error {
	return nil
}
func (m *mockUserRepo) AdminUpdateUser(_ context.Context, _ int64, _, _ string) error {
	return nil
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (int64, error) {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return 0, nil
}
func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, nil
}
func (m *mockUserRepo) SetVerified(ctx context.Context, userID int64, isVerified bool) error {
	if m.setVerifiedFn != nil {
		return m.setVerifiedFn(ctx, userID, isVerified)
	}
	return nil
}

func (m *mockUserRepo) UpdateAvatarURL(_ context.Context, _ int64, _ string) error { return nil }
func (m *mockUserRepo) UpdateProfile(_ context.Context, _ *domain.User) error      { return nil }
func (m *mockUserRepo) UpdatePassword(_ context.Context, _ int64, _ string) error  { return nil }
func (m *mockUserRepo) UpdateEmail(_ context.Context, _ int64, _ string) error     { return nil }
func (m *mockUserRepo) ListAll(_ context.Context, _, _ int, _ string, _ *bool) ([]domain.User, int, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) SetBanned(_ context.Context, _ int64, _ bool) error { return nil }
func (m *mockUserRepo) DeleteUnverifiedBefore(ctx context.Context, before time.Time) (int64, error) {
	if m.deleteUnverifiedBeforeFn != nil {
		return m.deleteUnverifiedBeforeFn(ctx, before)
	}
	return 0, nil
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
	if m.createFn != nil {
		return m.createFn(ctx, session)
	}
	return nil
}
func (m *mockSessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockSessionRepo) DeleteByID(ctx context.Context, id string) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	return nil
}

type mockMailService struct{}

func (m *mockMailService) SendVerification(email, token string) error {
	return nil
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
		&mockMailService{},
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

func TestRegister_ShortPassword(t *testing.T) {
	uc := newUsecase(&mockUserRepo{}, &mockSessionRepo{}, &mockVerificationRepo{})
	_, err := uc.Register(context.Background(), "user", "test@example.com", "short")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestRegister_InvalidUsername(t *testing.T) {
	uc := newUsecase(&mockUserRepo{}, &mockSessionRepo{}, &mockVerificationRepo{})
	_, err := uc.Register(context.Background(), "a!", "test@example.com", "password123")
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
			return &domain.User{ID: 1, PasswordHash: string(hash), IsVerified: true}, nil
		},
	}
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})

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
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})

	_, _, err := uc.Login(context.Background(), "no@example.com", "password123")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestLogin_NotVerified(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	userRepo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, PasswordHash: string(hash), IsVerified: false}, nil
		},
	}
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})

	_, _, err := uc.Login(context.Background(), "test@example.com", "password123")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
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
		deleteByIDFn: func(ctx context.Context, id string) error { return nil },
	}
	uc := newUsecase(&mockUserRepo{}, sessionRepo, &mockVerificationRepo{})
	if err := uc.Logout(context.Background(), "some-session"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSession_Success(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return &domain.Session{ID: id, UserID: 1, ExpiresAt: time.Now().Add(time.Hour)}, nil
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
			return &domain.Session{ID: id, UserID: 1, ExpiresAt: time.Now().Add(-time.Hour)}, nil
		},
		deleteByIDFn: func(ctx context.Context, id string) error { return nil },
	}
	uc := newUsecase(&mockUserRepo{}, sessionRepo, &mockVerificationRepo{})

	_, err := uc.ValidateSession(context.Background(), "expired")
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
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
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}

func TestValidateSession_ExpiredFromRepository(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return nil, domain.ErrSessionExpired
		},
	}
	uc := newUsecase(&mockUserRepo{}, sessionRepo, &mockVerificationRepo{})

	_, err := uc.ValidateSession(context.Background(), "expired-session")
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}

func TestValidateSession_UserNotFound(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id string) (*domain.Session, error) {
			return &domain.Session{ID: id, UserID: 1, ExpiresAt: time.Now().Add(time.Hour)}, nil
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

func TestConfirmEmail_Success(t *testing.T) {
	verificationRepo := &mockVerificationRepo{
		getFn: func(ctx context.Context, token string) (*domain.VerificationToken, error) {
			return &domain.VerificationToken{ID: 1, UserID: 1, Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}
	userRepo := &mockUserRepo{
		setVerifiedFn: func(ctx context.Context, userID int64, isVerified bool) error { return nil },
	}
	uc := newUsecase(userRepo, &mockSessionRepo{}, verificationRepo)

	if err := uc.ConfirmEmail(context.Background(), "valid-token"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfirmEmail_TokenNotFound(t *testing.T) {
	verificationRepo := &mockVerificationRepo{
		getFn: func(ctx context.Context, token string) (*domain.VerificationToken, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := newUsecase(&mockUserRepo{}, &mockSessionRepo{}, verificationRepo)

	err := uc.ConfirmEmail(context.Background(), "bad-token")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestConfirmEmail_TokenExpired(t *testing.T) {
	verificationRepo := &mockVerificationRepo{
		getFn: func(ctx context.Context, token string) (*domain.VerificationToken, error) {
			return &domain.VerificationToken{ID: 1, UserID: 1, Token: token, ExpiresAt: time.Now().Add(-time.Hour)}, nil
		},
	}
	uc := newUsecase(&mockUserRepo{}, &mockSessionRepo{}, verificationRepo)

	err := uc.ConfirmEmail(context.Background(), "expired-token")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestCleanupUnverified_DeletesUsers(t *testing.T) {
	var calledWith time.Time
	userRepo := &mockUserRepo{
		deleteUnverifiedBeforeFn: func(_ context.Context, before time.Time) (int64, error) {
			calledWith = before
			return 3, nil
		},
	}
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})

	uc.CleanupUnverified(context.Background())

	if calledWith.IsZero() {
		t.Fatal("DeleteUnverifiedBefore was not called")
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	if calledWith.Sub(cutoff) > time.Second || cutoff.Sub(calledWith) > time.Second {
		t.Errorf("cutoff should be ~24h ago, got %v", calledWith)
	}
}

func TestCleanupUnverified_HandlesError(t *testing.T) {
	userRepo := &mockUserRepo{
		deleteUnverifiedBeforeFn: func(_ context.Context, _ time.Time) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	uc := newUsecase(userRepo, &mockSessionRepo{}, &mockVerificationRepo{})

	uc.CleanupUnverified(context.Background())
}
