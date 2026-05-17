package grpcutil

import (
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestDomainToGRPCError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"nil", nil, codes.OK},
		{"not_found", domain.ErrNotFound, codes.NotFound},
		{"unauthorized", domain.ErrUnauthorized, codes.Unauthenticated},
		{"session_expired", domain.ErrSessionExpired, codes.Unauthenticated},
		{"conflict", domain.ErrConflict, codes.AlreadyExists},
		{"invalid_input", domain.ErrInvalidInput, codes.InvalidArgument},
		{"forbidden", domain.ErrForbidden, codes.PermissionDenied},
		{"service_unavailable", domain.ErrServiceUnavailable, codes.ResourceExhausted},
		{"wrapped_not_found", fmt.Errorf("wrap: %w", domain.ErrNotFound), codes.NotFound},
		{"unknown", fmt.Errorf("something broke"), codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DomainToGRPCError(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Fatalf("want nil, got %v", got)
				}
				return
			}
			st, ok := status.FromError(got)
			if !ok {
				t.Fatalf("expected gRPC status error, got %v", got)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("want code %v, got %v", tt.wantCode, st.Code())
			}
		})
	}
}

func TestGRPCToDomainError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"nil", nil, nil},
		{"not_found", status.Error(codes.NotFound, "x"), domain.ErrNotFound},
		{"unauthenticated", status.Error(codes.Unauthenticated, "x"), domain.ErrUnauthorized},
		{"already_exists", status.Error(codes.AlreadyExists, "x"), domain.ErrConflict},
		{"invalid_argument", status.Error(codes.InvalidArgument, "x"), domain.ErrInvalidInput},
		{"permission_denied", status.Error(codes.PermissionDenied, "x"), domain.ErrForbidden},
		{"resource_exhausted", status.Error(codes.ResourceExhausted, "x"), domain.ErrServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GRPCToDomainError(tt.err)
			if tt.wantErr == nil {
				if got != nil {
					t.Fatalf("want nil, got %v", got)
				}
				return
			}
			if got != tt.wantErr {
				t.Errorf("want %v, got %v", tt.wantErr, got)
			}
		})
	}
}
