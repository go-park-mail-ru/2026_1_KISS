package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newYandexTestProvider(t *testing.T, mux *http.ServeMux) (*YandexProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	p := NewYandexProvider("yid", "ysecret", "https://app.example/cb", srv.Client())
	p.authURL = srv.URL + "/authorize"
	p.tokenURL = srv.URL + "/token"
	p.userInfoURL = srv.URL + "/info"
	return p, srv
}

func TestYandexProvider_AuthorizationURL(t *testing.T) {
	p, _ := newYandexTestProvider(t, http.NewServeMux())
	got := p.AuthorizationURL("s", "c")
	for _, want := range []string{"client_id=yid", "response_type=code", "code_challenge_method=S256", "state=s", "code_challenge=c"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestYandexProvider_Exchange_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya-tok","token_type":"bearer"}`))
	})
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "OAuth ya-tok" {
			t.Errorf("expected 'OAuth ya-tok', got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42","login":"vasya","default_email":"v@ya.ru","display_name":"Vasya","default_avatar_id":"av","is_avatar_empty":false}`))
	})
	p, _ := newYandexTestProvider(t, mux)

	info, err := p.Exchange(context.Background(), "code", "verifier", "")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if info.ProviderID != "42" || info.Email != "v@ya.ru" || info.Username != "Vasya" ||
		info.AvatarURL != "https://avatars.yandex.net/get-yapic/av/islands-200" || !info.EmailVerified {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestYandexProvider_Exchange_NoAvatar(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"t"}`))
	})
	mux.HandleFunc("/info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"1","login":"l","default_email":"e@e.ru","is_avatar_empty":true,"default_avatar_id":"x"}`))
	})
	p, _ := newYandexTestProvider(t, mux)

	info, err := p.Exchange(context.Background(), "c", "v", "")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if info.AvatarURL != "" {
		t.Errorf("expected empty avatar when is_avatar_empty=true, got %q", info.AvatarURL)
	}
}
