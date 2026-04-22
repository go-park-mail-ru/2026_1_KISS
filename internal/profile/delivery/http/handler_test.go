package http_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	profilehttp "github.com/go-park-mail-ru/2026_1_KISS/internal/profile/delivery/http"
)

type mockProfileUsecase struct {
	uploadAvatarFn   func(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, contentType string) (*domain.User, error)
	updateProfileFn  func(ctx context.Context, userID int64, username, status, description string) (*domain.User, error)
	changePasswordFn func(ctx context.Context, userID int64, currentPassword, newPassword string) error
	changeEmailFn    func(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error)
}

func (m *mockProfileUsecase) UploadAvatar(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, contentType string) (*domain.User, error) {
	if m.uploadAvatarFn != nil {
		return m.uploadAvatarFn(ctx, userID, file, fileSize, contentType)
	}
	return nil, nil
}

func (m *mockProfileUsecase) UpdateProfile(ctx context.Context, userID int64, username, status, description string) (*domain.User, error) {
	if m.updateProfileFn != nil {
		return m.updateProfileFn(ctx, userID, username, status, description)
	}
	return nil, nil
}

func (m *mockProfileUsecase) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	if m.changePasswordFn != nil {
		return m.changePasswordFn(ctx, userID, currentPassword, newPassword)
	}
	return nil
}

func (m *mockProfileUsecase) ChangeEmail(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error) {
	if m.changeEmailFn != nil {
		return m.changeEmailFn(ctx, userID, newEmail, password)
	}
	return nil, nil
}

func reqWithUser(req *http.Request, user *domain.User) *http.Request {
	ctx := middleware.SetUserInContext(req.Context(), user)
	return req.WithContext(ctx)
}

func testUser() *domain.User {
	return &domain.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestUploadAvatar_Handler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := testUser()
		h := profilehttp.New(&mockProfileUsecase{
			uploadAvatarFn: func(_ context.Context, _ int64, _ io.ReadSeeker, _ int64, _ string) (*domain.User, error) {
				u := testUser()
				u.AvatarURL = "/uploads/new.jpg"
				return u, nil
			},
		}, 5<<20)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, _ := writer.CreateFormFile("avatar", "test.jpg")
		_, _ = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/users/me/avatar", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = reqWithUser(req, user)
		rec := httptest.NewRecorder()

		h.UploadAvatar(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("no file field", func(t *testing.T) {
		user := testUser()
		h := profilehttp.New(&mockProfileUsecase{}, 5<<20)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/users/me/avatar", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = reqWithUser(req, user)
		rec := httptest.NewRecorder()

		h.UploadAvatar(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("want 400, got %d", rec.Code)
		}
	})

	t.Run("usecase error", func(t *testing.T) {
		user := testUser()
		h := profilehttp.New(&mockProfileUsecase{
			uploadAvatarFn: func(_ context.Context, _ int64, _ io.ReadSeeker, _ int64, _ string) (*domain.User, error) {
				return nil, domain.ErrInvalidInput
			},
		}, 5<<20)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, _ := writer.CreateFormFile("avatar", "test.txt")
		_, _ = part.Write([]byte("text content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/users/me/avatar", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = reqWithUser(req, user)
		rec := httptest.NewRecorder()

		h.UploadAvatar(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("want 400, got %d", rec.Code)
		}
	})
}

func TestUpdateProfile_Handler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(ctx context.Context, userID int64, username, status, description string) (*domain.User, error)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"username":"newuser","status":"hi","description":"bio"}`,
			mockFn: func(_ context.Context, _ int64, _ string, _ string, _ string) (*domain.User, error) {
				return testUser(), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "conflict",
			body: `{"username":"taken","status":"","description":""}`,
			mockFn: func(_ context.Context, _ int64, _ string, _ string, _ string) (*domain.User, error) {
				return nil, domain.ErrConflict
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := profilehttp.New(&mockProfileUsecase{updateProfileFn: tc.mockFn}, 5<<20)
			req := httptest.NewRequest("PUT", "/api/v1/users/me", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = reqWithUser(req, testUser())
			rec := httptest.NewRecorder()

			h.UpdateProfile(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("want %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestChangePassword_Handler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(ctx context.Context, userID int64, currentPassword, newPassword string) error
		wantStatus int
	}{
		{
			name: "success",
			body: `{"current_password":"old123456","new_password":"new123456"}`,
			mockFn: func(_ context.Context, _ int64, _ string, _ string) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "wrong password",
			body: `{"current_password":"wrong","new_password":"new123456"}`,
			mockFn: func(_ context.Context, _ int64, _ string, _ string) error {
				return domain.ErrUnauthorized
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := profilehttp.New(&mockProfileUsecase{changePasswordFn: tc.mockFn}, 5<<20)
			req := httptest.NewRequest("PUT", "/api/v1/users/me/password", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = reqWithUser(req, testUser())
			rec := httptest.NewRecorder()

			h.ChangePassword(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("want %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestChangeEmail_Handler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"new_email":"new@example.com","password":"pass1234"}`,
			mockFn: func(_ context.Context, _ int64, _ string, _ string) (*domain.User, error) {
				return testUser(), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "conflict",
			body: `{"new_email":"taken@example.com","password":"pass1234"}`,
			mockFn: func(_ context.Context, _ int64, _ string, _ string) (*domain.User, error) {
				return nil, domain.ErrConflict
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := profilehttp.New(&mockProfileUsecase{changeEmailFn: tc.mockFn}, 5<<20)
			req := httptest.NewRequest("PUT", "/api/v1/users/me/email", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = reqWithUser(req, testUser())
			rec := httptest.NewRecorder()

			h.ChangeEmail(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("want %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRegisterRoutes_Profile(t *testing.T) {
	h := profilehttp.New(&mockProfileUsecase{}, 5<<20)
	mux := http.NewServeMux()
	authMw := func(next http.Handler) http.Handler { return next }
	h.RegisterRoutes(mux, authMw)
}
