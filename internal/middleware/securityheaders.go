package middleware

import "net/http"

func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", "default-src 'none'")
			w.Header().Set("X-XSS-Protection", "0")
			next.ServeHTTP(w, r)
		})
	}
}
