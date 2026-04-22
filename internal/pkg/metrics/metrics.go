package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var HTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	},
	[]string{"method", "path", "status"},
)

var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path", "status"},
)

var GRPCRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "grpc_requests_total",
		Help: "Total number of gRPC requests.",
	},
	[]string{"service", "method", "code"},
)

var GRPCRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "grpc_request_duration_seconds",
		Help:    "gRPC request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"service", "method", "code"},
)

func init() {
	prometheus.MustRegister(HTTPRequestsTotal, HTTPRequestDuration)
	prometheus.MustRegister(GRPCRequestsTotal, GRPCRequestDuration)
}

func StartMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()
	slog.Info("metrics server started", "addr", addr)
	return srv
}

func ShutdownMetricsServer(srv *http.Server) {
	if srv == nil {
		return
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		slog.Error("metrics server shutdown error", "error", err)
	}
}
