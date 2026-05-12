package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newGoogleTestProvider(t *testing.T, mux *http.ServeMux) (*GoogleProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	p := NewGoogleProvider("client-id", "client-secret", "https://app.example/cb", srv.Client())
	p.authURL = srv.URL + "/auth"
	p.tokenURL = srv.URL + "/token"
	p.userInfoURL = srv.URL + "/userinfo"
	return p, srv
}

func TestGoogleProvider_AuthorizationURL(t *testing.T) {
	p, _ := newGoogleTestProvider(t, http.NewServeMux())
	got := p.AuthorizationURL("state-1", "challenge-1")
	for _, want := range []string{
		"client_id=client-id",
		"redirect_uri=https%3A%2F%2Fapp.example%2Fcb",
		"response_type=code",
		"scope=openid+email+profile",
		"state=state-1",
		"code_challenge=challenge-1",
		"code_challenge_method=S256",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("AuthorizationURL missing %q in %q", want, got)
		}
	}
}

func TestGoogleProvider_Exchange_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("code"); got != "the-code" {
			t.Errorf("token: code=%q", got)
		}
		if got := r.FormValue("code_verifier"); got != "the-verifier" {
			t.Errorf("token: code_verifier=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","token_type":"Bearer"}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer abc" {
			t.Errorf("userinfo: bad auth header %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"google-123","email":"u@example.com","email_verified":true,"given_name":"Ivan","picture":"https://pic"}`))
	})
	p, _ := newGoogleTestProvider(t, mux)

	info, err := p.Exchange(context.Background(), "the-code", "the-verifier", "")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if info.ProviderID != "google-123" || info.Email != "u@example.com" ||
		!info.EmailVerified || info.Username != "Ivan" || info.AvatarURL != "https://pic" {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestGoogleProvider_Exchange_TokenError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})
	p, _ := newGoogleTestProvider(t, mux)

	if _, err := p.Exchange(context.Background(), "x", "y", ""); err == nil {
		t.Fatalf("expected error on bad token response")
	}
}
