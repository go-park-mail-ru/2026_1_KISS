package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type authUsecase interface {
	Register(ctx context.Context, username, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*domain.Session, *domain.User, error)
	Logout(ctx context.Context, sessionID string) error
	ValidateSession(ctx context.Context, sessionID string) (*domain.User, error)
	ConfirmEmail(ctx context.Context, token string) error
}

type AuthHandler struct {
	usecase authUsecase
}

func New(uc authUsecase) *AuthHandler {
	return &AuthHandler{usecase: uc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.usecase.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, NewUserResponse(user))
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.Me)
	mux.HandleFunc("GET /api/v1/auth/confirm", h.ConfirmEmail)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, user, err := h.usecase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	httputil.JSON(w, http.StatusOK, NewUserResponse(user))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		httputil.JSON(w, http.StatusOK, nil)
		return
	}

	_ = h.usecase.Logout(r.Context(), cookie.Value)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	httputil.JSON(w, http.StatusOK, nil)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.usecase.ValidateSession(r.Context(), cookie.Value)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, NewUserResponse(user))
}

func mapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		httputil.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		httputil.Error(w, http.StatusConflict, "email or username already exists")
	case errors.Is(err, domain.ErrUnauthorized):
		httputil.Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, domain.ErrInvalidInput):
		httputil.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		httputil.Error(w, http.StatusForbidden, "access denied")
	default:
		httputil.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *AuthHandler) ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	if token == "" {
		httputil.Error(w, http.StatusBadRequest, "token required")
		return
	}

	if err := h.usecase.ConfirmEmail(r.Context(), token); err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"message": "email confirmed",
	})
}
