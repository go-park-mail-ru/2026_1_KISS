package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// Hijack делегирует вызов нижестоящему ResponseWriter, чтобы апгрейд
// HTTP→WebSocket работал через цепочку middleware (websocket.Accept
// требует http.Hijacker).
func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := sr.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("statusRecorder: underlying ResponseWriter does not implement http.Hijacker")
	}
	return hj.Hijack()
}

func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(sr, r)
			duration := time.Since(start)
			logger.Info(r.Context(), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sr.statusCode,
				"duration", duration.String(),
			)
		})
	}
}
