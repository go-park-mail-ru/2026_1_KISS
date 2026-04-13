package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type visitor struct {
	tokens   int
	lastSeen time.Time
}

func RateLimit(maxRequests int, window time.Duration) Middleware {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)

	go func() {
		for {
			time.Sleep(window)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > window*2 {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				v = &visitor{tokens: maxRequests}
				visitors[ip] = v
			}

			now := time.Now()
			elapsed := now.Sub(v.lastSeen)
			if elapsed >= window {
				v.tokens = maxRequests
			}
			v.lastSeen = now

			if v.tokens <= 0 {
				mu.Unlock()
				httputil.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			v.tokens--
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
