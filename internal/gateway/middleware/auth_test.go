package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

func TestAuth_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().ValidateSession(gomock.Any(), gomock.Any()).Return(&pb.ValidateSessionResponse{
		User: &pb.UserInfo{Id: 1, Username: "testuser"},
	}, nil)

	mw := Auth(client)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			t.Fatal("expected user in context")
		}
		if user.ID != 1 {
			t.Errorf("want user id 1, got %d", user.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-123"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAuth_NoCookie(t *testing.T) {
	mw := Auth(nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestAuth_InvalidSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().ValidateSession(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.Unauthenticated, "expired"))

	mw := Auth(client)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "expired"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}
