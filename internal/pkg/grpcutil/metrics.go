package grpcutil

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
)

func MetricsUnaryInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()

		st, _ := status.FromError(err)
		code := st.Code().String()

		metrics.GRPCRequestsTotal.WithLabelValues(serviceName, info.FullMethod, code).Inc()
		metrics.GRPCRequestDuration.WithLabelValues(serviceName, info.FullMethod, code).Observe(duration)

		return resp, err
	}
}
