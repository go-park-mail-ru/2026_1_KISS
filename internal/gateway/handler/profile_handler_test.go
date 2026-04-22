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

func TestProfileHandler_UpdateProfile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(&pb.UserResponse{
		User: &pb.UserInfo{Id: 1, Username: "newname", Status: "active"},
	}, nil)

	h := NewProfileHandler(client, 2*1024*1024)
	body, _ := json.Marshal(updateProfileRequest{Username: "newname", Status: "active"})
	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestProfileHandler_ChangePassword_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().ChangePassword(gomock.Any(), gomock.Any()).Return(&pb.ChangePasswordResponse{}, nil)

	h := NewProfileHandler(client, 2*1024*1024)
	body, _ := json.Marshal(changePasswordRequest{CurrentPassword: "old", NewPassword: "New123!!"})
	req := httptest.NewRequest("PUT", "/api/v1/users/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ChangePassword(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestProfileHandler_ChangeEmail_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().ChangeEmail(gomock.Any(), gomock.Any()).Return(&pb.UserResponse{
		User: &pb.UserInfo{Id: 1, Email: "new@example.com"},
	}, nil)

	h := NewProfileHandler(client, 2*1024*1024)
	body, _ := json.Marshal(changeEmailRequest{NewEmail: "new@example.com", Password: "Pass123!"})
	req := httptest.NewRequest("PUT", "/api/v1/users/me/email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ChangeEmail(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestProfileHandler_ChangePassword_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().ChangePassword(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.Unauthenticated, "wrong password"))

	h := NewProfileHandler(client, 2*1024*1024)
	body, _ := json.Marshal(changePasswordRequest{CurrentPassword: "wrong", NewPassword: "New123!!"})
	req := httptest.NewRequest("PUT", "/api/v1/users/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ChangePassword(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestProfileHandler_Unauthorized(t *testing.T) {
	h := NewProfileHandler(nil, 2*1024*1024)
	body, _ := json.Marshal(updateProfileRequest{Username: "test"})
	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}
