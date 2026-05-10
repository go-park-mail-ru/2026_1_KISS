package grpcutil

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func DomainToGRPCError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrSessionExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrPaymentFailed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrYooKassaUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func GRPCToDomainError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.NotFound:
		return domain.ErrNotFound
	case codes.Unauthenticated:
		if strings.Contains(st.Message(), "session expired") {
			return domain.ErrSessionExpired
		}
		return domain.ErrUnauthorized
	case codes.AlreadyExists:
		return domain.ErrConflict
	case codes.InvalidArgument:
		return domain.ErrInvalidInput
	case codes.PermissionDenied:
		return domain.ErrForbidden
	case codes.FailedPrecondition:
		return domain.ErrPaymentFailed
	case codes.Unavailable:
		return domain.ErrYooKassaUnavailable
	default:
		return err
	}
}
