package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/dto"
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
	appURL  string
}

func New(uc authUsecase, appURL string) *AuthHandler {
	return &AuthHandler{usecase: uc, appURL: appURL}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.Me)
	mux.HandleFunc("GET /api/v1/auth/confirm", h.ConfirmEmail)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.usecase.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, dto.NewUserResponse(user))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
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

	middleware.SetCSRFCookie(w, session.ExpiresAt, false)

	httputil.JSON(w, http.StatusOK, dto.NewUserResponse(user))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		httputil.JSON(w, http.StatusOK, nil)
		return
	}

	_ = h.usecase.Logout(r.Context(), cookie.Value)
	clearSessionCookie(w)
	middleware.ClearCSRFCookie(w)

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
		if errors.Is(err, domain.ErrSessionExpired) {
			clearSessionCookie(w)
		}
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.NewUserResponse(user))
}

func (h *AuthHandler) ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, h.appURL+"/login?error=invalid_token", http.StatusSeeOther)
		return
	}

	if err := h.usecase.ConfirmEmail(r.Context(), token); err != nil {
		http.Redirect(w, r, h.appURL+"/login?error=invalid_token", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, h.appURL+"/login?verified=1", http.StatusSeeOther)
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func mapDomainError(w http.ResponseWriter, err error) {
	httputil.MapDomainError(w, err)
}

// ensure time import is used if needed elsewhere
var _ = time.Time{}
