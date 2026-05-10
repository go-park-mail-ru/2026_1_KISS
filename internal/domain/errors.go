package domain

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrSessionExpired      = errors.New("session expired")
	ErrConflict            = errors.New("conflict")
	ErrInvalidInput        = errors.New("invalid input")
	ErrForbidden           = errors.New("forbidden")
	ErrPaymentFailed       = errors.New("payment failed")
	ErrYooKassaUnavailable = errors.New("yookassa unavailable")
)
