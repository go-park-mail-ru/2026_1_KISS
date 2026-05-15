package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newVKIDTestProvider(t *testing.T, mux *http.ServeMux) (*VKIDProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	p := NewVKIDProvider("vkid", "vksecret", "https://app.example/cb", srv.Client())
	p.authURL = srv.URL + "/authorize"
	p.tokenURL = srv.URL + "/token"
	p.userInfoURL = srv.URL + "/user_info"
	return p, srv
}

func TestVKIDProvider_AuthorizationURL(t *testing.T) {
	p, _ := newVKIDTestProvider(t, http.NewServeMux())
	got := p.AuthorizationURL("st", "ch")
	for _, want := range []string{"client_id=vkid", "code_challenge_method=S256", "scope=email", "state=st", "code_challenge=ch"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestVKIDProvider_Exchange_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("code_verifier") == "" {
			t.Error("PKCE verifier must be sent")
		}
		_, _ = w.Write([]byte(`{"access_token":"vk-tok","user_id":777}`))
	})
	mux.HandleFunc("/user_info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"user_id":"777","first_name":"Petya","email":"p@vk.com","avatar":"https://vk/p"}}`))
	})
	p, _ := newVKIDTestProvider(t, mux)

	info, err := p.Exchange(context.Background(), "c", "v", "")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if info.ProviderID != "777" || info.Email != "p@vk.com" || info.Username != "Petya" || !info.EmailVerified {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestVKIDProvider_Exchange_NoEmail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"t","user_id":1}`))
	})
	mux.HandleFunc("/user_info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"user_id":"1","first_name":"N"}}`))
	})
	p, _ := newVKIDTestProvider(t, mux)

	info, err := p.Exchange(context.Background(), "c", "v", "")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if info.Email != "" || info.EmailVerified {
		t.Errorf("expected empty unverified email, got %+v", info)
	}
}

func TestVKIDProvider_Exchange_EmptyAccessTokenDumpsBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"invalid_request","error_description":"missing device_id"}`))
	})
	p, _ := newVKIDTestProvider(t, mux)

	_, err := p.Exchange(context.Background(), "c", "v", "")
	if err == nil {
		t.Fatal("expected error on empty access_token")
	}
	if !strings.Contains(err.Error(), "missing device_id") {
		t.Errorf("expected error to include response body, got: %v", err)
	}
}

func TestVKIDProvider_Exchange_PassesDeviceID(t *testing.T) {
	var captured string
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		captured = r.FormValue("device_id")
		_, _ = w.Write([]byte(`{"access_token":"t","user_id":1}`))
	})
	mux.HandleFunc("/user_info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"user_id":"1","first_name":"X"}}`))
	})
	p, _ := newVKIDTestProvider(t, mux)

	if _, err := p.Exchange(context.Background(), "c", "v", "dev-42"); err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if captured != "dev-42" {
		t.Errorf("expected device_id=dev-42 in token form, got %q", captured)
	}
}

func TestVKIDProvider_Exchange_FallbackUserIDFromToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"t","user_id":555}`))
	})
	mux.HandleFunc("/user_info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"first_name":"X"}}`))
	})
	p, _ := newVKIDTestProvider(t, mux)

	info, err := p.Exchange(context.Background(), "c", "v", "")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if info.ProviderID != "555" {
		t.Errorf("expected provider_id fallback to token.user_id=555, got %q", info.ProviderID)
	}
}
