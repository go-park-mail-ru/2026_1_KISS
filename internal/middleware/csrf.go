package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func SetCSRFCookie(w http.ResponseWriter, expiresAt interface{ Unix() int64 }) string {
	token, err := generateCSRFToken()
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

func ClearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
	})
}

func CSRF(skipPaths map[string]bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie("csrf_token")
			if err != nil {
				httputil.Error(w, http.StatusForbidden, "csrf token missing")
				return
			}

			header := r.Header.Get("X-CSRF-Token")
			if header == "" || header != cookie.Value {
				httputil.Error(w, http.StatusForbidden, "csrf token invalid")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
