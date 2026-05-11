package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

func newOAuthRequest(provider, query string) *http.Request {
	target := "/api/v1/auth/oauth/" + provider + "/callback"
	if query != "" {
		target += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.SetPathValue("provider", provider)
	return req
}

func TestOAuthHandler_Start_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	client.EXPECT().OAuthStart(gomock.Any(), &pb.OAuthStartRequest{Provider: "google"}).
		Return(&pb.OAuthStartResponse{AuthUrl: "https://accounts.google.com/auth?x=1", State: "st-1", ExpiresAt: 9999999999}, nil)

	h := NewOAuthHandler(client, false, "http://localhost:3000")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/start", nil)
	req.SetPathValue("provider", "google")
	rec := httptest.NewRecorder()

	h.Start(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("want 307, got %d", rec.Code)
	}
	loc := rec.Result().Header.Get("Location")
	if !strings.HasPrefix(loc, "https://accounts.google.com/auth") {
		t.Errorf("unexpected Location: %q", loc)
	}
	var stateCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
	}
	if stateCookie == nil || stateCookie.Value != "st-1" {
		t.Fatalf("oauth_state cookie missing or wrong value: %+v", stateCookie)
	}
}

func TestOAuthHandler_Start_UnknownProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	h := NewOAuthHandler(client, false, "http://localhost:3000")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/foo/start", nil)
	req.SetPathValue("provider", "foo")
	rec := httptest.NewRecorder()

	h.Start(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("want 303 redirect to error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Result().Header.Get("Location"), "oauth_error=unknown_provider") {
		t.Errorf("expected oauth_error=unknown_provider, got %q", rec.Result().Header.Get("Location"))
	}
}

func TestOAuthHandler_Callback_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	client.EXPECT().OAuthCallback(gomock.Any(), &pb.OAuthCallbackRequest{Provider: "google", Code: "c", State: "st-1"}).
		Return(&pb.LoginResponse{SessionId: "sess-1", ExpiresAt: 9999999999, User: &pb.UserInfo{Id: 1}}, nil)

	h := NewOAuthHandler(client, false, "http://localhost:3000")
	req := newOAuthRequest("google", "code=c&state=st-1")
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "st-1"})
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("want 303, got %d", rec.Code)
	}
	if rec.Result().Header.Get("Location") != "http://localhost:3000/files" {
		t.Errorf("unexpected Location: %q", rec.Result().Header.Get("Location"))
	}
	var sessionFound bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" && c.Value == "sess-1" {
			sessionFound = true
		}
	}
	if !sessionFound {
		t.Errorf("session_id cookie not set")
	}
}

func TestOAuthHandler_Callback_ProviderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	h := NewOAuthHandler(client, false, "http://localhost:3000")

	req := newOAuthRequest("google", "error=access_denied")
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("want 303, got %d", rec.Code)
	}
	if !strings.Contains(rec.Result().Header.Get("Location"), "oauth_error=denied") {
		t.Errorf("expected oauth_error=denied, got %q", rec.Result().Header.Get("Location"))
	}
}

func TestOAuthHandler_Callback_StateMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	h := NewOAuthHandler(client, false, "http://localhost:3000")

	req := newOAuthRequest("google", "code=c&state=from-provider")
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "from-cookie"})
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if !strings.Contains(rec.Result().Header.Get("Location"), "oauth_error=invalid_state") {
		t.Errorf("expected oauth_error=invalid_state, got %q", rec.Result().Header.Get("Location"))
	}
}

func TestOAuthHandler_Callback_MissingCodeOrState(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	h := NewOAuthHandler(client, false, "http://localhost:3000")

	req := newOAuthRequest("google", "")
	rec := httptest.NewRecorder()
	h.Callback(rec, req)
	if !strings.Contains(rec.Result().Header.Get("Location"), "oauth_error=invalid_request") {
		t.Errorf("expected oauth_error=invalid_request, got %q", rec.Result().Header.Get("Location"))
	}
}

func TestOAuthHandler_Callback_UsecaseConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	client.EXPECT().OAuthCallback(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.AlreadyExists, "email taken"))

	h := NewOAuthHandler(client, false, "http://localhost:3000")
	req := newOAuthRequest("google", "code=c&state=st-1")
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "st-1"})
	rec := httptest.NewRecorder()

	h.Callback(rec, req)
	if !strings.Contains(rec.Result().Header.Get("Location"), "oauth_error=email_taken") {
		t.Errorf("expected oauth_error=email_taken, got %q", rec.Result().Header.Get("Location"))
	}
}

func TestOAuthHandler_RegisterRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	h := NewOAuthHandler(client, false, "http://x")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	if mux == nil {
		t.Fatal("mux should not be nil")
	}
}
