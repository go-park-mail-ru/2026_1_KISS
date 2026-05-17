package httputil

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func MapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		Error(w, http.StatusConflict, "email or username already exists")
	case errors.Is(err, domain.ErrSessionExpired):
		Error(w, http.StatusUnauthorized, "session expired")
	case errors.Is(err, domain.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, domain.ErrInvalidInput):
		Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		Error(w, http.StatusForbidden, "access denied")
	case errors.Is(err, domain.ErrPaymentFailed):
		Error(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrYooKassaUnavailable):
		Error(w, http.StatusBadGateway, "payment provider unavailable")
	case errors.Is(err, domain.ErrServiceUnavailable):
		Error(w, http.StatusServiceUnavailable, err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}
