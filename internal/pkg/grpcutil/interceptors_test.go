package grpcutil

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoggingUnaryInterceptor(t *testing.T) {
	interceptor := LoggingUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	resp, err := interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("want ok, got %v", resp)
	}
}

func TestLoggingUnaryInterceptor_Error(t *testing.T) {
	interceptor := LoggingUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	_, err := interceptor(context.Background(), nil, info, handler)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRecoveryUnaryInterceptor(t *testing.T) {
	interceptor := RecoveryUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	_, err := interceptor(context.Background(), nil, info, handler)
	if err == nil {
		t.Fatal("expected error after panic")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("want Internal, got %v", st.Code())
	}
}

func TestRecoveryUnaryInterceptor_NoPanic(t *testing.T) {
	interceptor := RecoveryUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	resp, err := interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("want ok, got %v", resp)
	}
}
