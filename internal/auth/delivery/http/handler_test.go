package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authhttp "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/delivery/http"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type mockAuthUsecase struct {
	registerFn        func(ctx context.Context, username, email, password string) (*domain.User, error)
	loginFn           func(ctx context.Context, email, password string) (*domain.Session, *domain.User, error)
	logoutFn          func(ctx context.Context, sessionID string) error
	validateSessionFn func(ctx context.Context, sessionID string) (*domain.User, error)
	confirmEmailFn    func(ctx context.Context, token string) error
}

func (m *mockAuthUsecase) Register(ctx context.Context, username, email, password string) (*domain.User, error) {
	if m.registerFn != nil {
		return m.registerFn(ctx, username, email, password)
	}
	return nil, nil
}

func (m *mockAuthUsecase) Login(ctx context.Context, email, password string) (*domain.Session, *domain.User, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, email, password)
	}
	return nil, nil, nil
}

func (m *mockAuthUsecase) Logout(ctx context.Context, sessionID string) error {
	if m.logoutFn != nil {
		return m.logoutFn(ctx, sessionID)
	}
	return nil
}

func (m *mockAuthUsecase) ValidateSession(ctx context.Context, sessionID string) (*domain.User, error) {
	if m.validateSessionFn != nil {
		return m.validateSessionFn(ctx, sessionID)
	}
	return nil, domain.ErrUnauthorized
}

func (m *mockAuthUsecase) ConfirmEmail(ctx context.Context, token string) error {
	if m.confirmEmailFn != nil {
		return m.confirmEmailFn(ctx, token)
	}
	return nil
}

type mockVerificationRepo struct {
	createFn func(ctx context.Context, v *domain.VerificationToken) error
	getFn    func(ctx context.Context, token string) (*domain.VerificationToken, error)
}

func (m *mockVerificationRepo) Create(ctx context.Context, v *domain.VerificationToken) error {
	if m.createFn != nil {
		return m.createFn(ctx, v)
	}
	return nil
}

func (m *mockVerificationRepo) Get(ctx context.Context, token string) (*domain.VerificationToken, error) {
	if m.getFn != nil {
		return m.getFn(ctx, token)
	}
	return nil, nil
}

type mockMailService struct {
	sendFn func(to, subject, body string) error
}

func (m *mockMailService) Send(to, subject, body string) error {
	if m.sendFn != nil {
		return m.sendFn(to, subject, body)
	}
	return nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(ctx context.Context, username, email, password string) (*domain.User, error)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"username":"testuser","email":"test@example.com","password":"password123"}`,
			mockFn: func(ctx context.Context, username, email, password string) (*domain.User, error) {
				return &domain.User{ID: 1, Username: username, Email: email, CreatedAt: time.Now()}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "conflict",
			body: `{"username":"testuser","email":"test@example.com","password":"password123"}`,
			mockFn: func(ctx context.Context, username, email, password string) (*domain.User, error) {
				return nil, domain.ErrConflict
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "invalid body",
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := authhttp.New(&mockAuthUsecase{registerFn: tc.mockFn})
			req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.Register(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("want status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(ctx context.Context, email, password string) (*domain.Session, *domain.User, error)
		wantStatus int
		wantCookie bool
	}{
		{
			name: "success",
			body: `{"email":"test@example.com","password":"password123"}`,
			mockFn: func(ctx context.Context, email, password string) (*domain.Session, *domain.User, error) {
				return &domain.Session{ID: "sess-1", ExpiresAt: time.Now().Add(time.Hour)},
					&domain.User{ID: 1, Email: email}, nil
			},
			wantStatus: http.StatusOK,
			wantCookie: true,
		},
		{
			name: "unauthorized",
			body: `{"email":"test@example.com","password":"wrong"}`,
			mockFn: func(ctx context.Context, email, password string) (*domain.Session, *domain.User, error) {
				return nil, nil, domain.ErrUnauthorized
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid body",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := authhttp.New(&mockAuthUsecase{loginFn: tc.mockFn})
			req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.Login(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("want status %d, got %d", tc.wantStatus, rec.Code)
			}
			if tc.wantCookie {
				found := false
				for _, c := range rec.Result().Cookies() {
					if c.Name == "session_id" {
						found = true
					}
				}
				if !found {
					t.Error("expected session_id cookie")
				}
			}
		})
	}
}

func TestLogout(t *testing.T) {
	t.Run("with cookie", func(t *testing.T) {
		h := authhttp.New(&mockAuthUsecase{
			logoutFn: func(ctx context.Context, sessionID string) error { return nil },
		})
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
		rec := httptest.NewRecorder()
		h.Logout(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("want 200, got %d", rec.Code)
		}
	})

	t.Run("without cookie", func(t *testing.T) {
		h := authhttp.New(&mockAuthUsecase{})
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		h.Logout(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("want 200, got %d", rec.Code)
		}
	})
}

func TestMe(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := authhttp.New(&mockAuthUsecase{
			validateSessionFn: func(ctx context.Context, sessionID string) (*domain.User, error) {
				return &domain.User{ID: 1, Username: "test", CreatedAt: time.Now()}, nil
			},
		})
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
		rec := httptest.NewRecorder()
		h.Me(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("want 200, got %d", rec.Code)
		}
	})

	t.Run("no cookie", func(t *testing.T) {
		h := authhttp.New(&mockAuthUsecase{})
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		h.Me(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", rec.Code)
		}
	})

	t.Run("invalid session", func(t *testing.T) {
		h := authhttp.New(&mockAuthUsecase{
			validateSessionFn: func(ctx context.Context, sessionID string) (*domain.User, error) {
				return nil, domain.ErrUnauthorized
			},
		})
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad"})
		rec := httptest.NewRecorder()
		h.Me(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", rec.Code)
		}
	})
}

func TestRegisterRoutes_Auth(t *testing.T) {
	h := authhttp.New(&mockAuthUsecase{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
}

func TestRegister_NotFound(t *testing.T) {
	h := authhttp.New(&mockAuthUsecase{
		registerFn: func(ctx context.Context, username, email, password string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"username":"u","email":"e@e.com","password":"pass1234"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestRegister_InvalidInput(t *testing.T) {
	h := authhttp.New(&mockAuthUsecase{
		registerFn: func(ctx context.Context, username, email, password string) (*domain.User, error) {
			return nil, domain.ErrInvalidInput
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"username":"u","email":"e@e.com","password":"pass1234"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestRegister_Forbidden(t *testing.T) {
	h := authhttp.New(&mockAuthUsecase{
		registerFn: func(ctx context.Context, username, email, password string) (*domain.User, error) {
			return nil, domain.ErrForbidden
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"username":"u","email":"e@e.com","password":"pass1234"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestRegister_InternalError(t *testing.T) {
	h := authhttp.New(&mockAuthUsecase{
		registerFn: func(ctx context.Context, username, email, password string) (*domain.User, error) {
			return nil, errors.New("unexpected")
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"username":"u","email":"e@e.com","password":"pass1234"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}
