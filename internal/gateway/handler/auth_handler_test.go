package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

func TestAuthHandler_Register_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().Register(gomock.Any(), gomock.Any()).Return(&pb.RegisterResponse{
		User: &pb.UserInfo{Id: 1, Username: "testuser", Email: "test@example.com"},
	}, nil)

	h := NewAuthHandler(client, false, "http://localhost:3000")
	body, _ := json.Marshal(registerRequest{Username: "testuser", Email: "test@example.com", Password: "Password123!"})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestAuthHandler_Register_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.AlreadyExists, "conflict"))

	h := NewAuthHandler(client, false, "http://localhost:3000")
	body, _ := json.Marshal(registerRequest{Username: "testuser", Email: "test@example.com", Password: "Password123!"})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().Login(gomock.Any(), gomock.Any()).Return(&pb.LoginResponse{
		SessionId: "sess-123",
		ExpiresAt: 1700000000,
		User:      &pb.UserInfo{Id: 1, Username: "testuser"},
	}, nil)

	h := NewAuthHandler(client, false, "http://localhost:3000")
	body, _ := json.Marshal(loginRequest{Email: "test@example.com", Password: "Password123!"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session_id" && c.Value == "sess-123" {
			found = true
		}
	}
	if !found {
		t.Error("session_id cookie not set")
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.Unauthenticated, "unauthorized"))

	h := NewAuthHandler(client, false, "http://localhost:3000")
	body, _ := json.Marshal(loginRequest{Email: "test@example.com", Password: "wrong"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(&pb.LogoutResponse{}, nil)

	h := NewAuthHandler(client, false, "http://localhost:3000")
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-123"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAuthHandler_Me_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().ValidateSession(gomock.Any(), gomock.Any()).Return(&pb.ValidateSessionResponse{
		User: &pb.UserInfo{Id: 1, Username: "testuser"},
	}, nil)

	h := NewAuthHandler(client, false, "http://localhost:3000")
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-123"})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	h := NewAuthHandler(nil, false, "http://localhost:3000")
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestAuthHandler_ConfirmEmail_NoToken(t *testing.T) {
	h := NewAuthHandler(nil, false, "http://localhost:3000")
	req := httptest.NewRequest("GET", "/api/v1/auth/confirm", nil)
	rec := httptest.NewRecorder()

	h.ConfirmEmail(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", rec.Code)
	}
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	h := NewAuthHandler(nil, false, "http://localhost:3000")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
}
