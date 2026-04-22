package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
)

func Metrics() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(sr, r)
			duration := time.Since(start).Seconds()

			status := strconv.Itoa(sr.statusCode)
			path := r.Pattern
			if path == "" {
				path = r.URL.Path
			}

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
		})
	}
}
