package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

const oauthStateCookieName = "oauth_state"

type OAuthHandler struct {
	client       pb.AuthServiceClient
	cookieSecure bool
	frontendURL  string
}

func NewOAuthHandler(client pb.AuthServiceClient, cookieSecure bool, frontendURL string) *OAuthHandler {
	return &OAuthHandler{client: client, cookieSecure: cookieSecure, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (h *OAuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/oauth/{provider}/start", h.Start)
	mux.HandleFunc("GET /api/v1/auth/oauth/{provider}/callback", h.Callback)
}

func (h *OAuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	if !domain.ValidOAuthProviders[providerName] {
		h.redirectError(w, r, "unknown_provider")
		return
	}

	resp, err := h.client.OAuthStart(r.Context(), &pb.OAuthStartRequest{Provider: providerName})
	if err != nil {
		domainErr := grpcutil.GRPCToDomainError(err)
		slog.Error("oauth start failed", "provider", providerName, "error", domainErr)
		h.redirectError(w, r, mapOAuthError(domainErr))
		return
	}

	expiresAt := time.Unix(resp.GetExpiresAt(), 0)
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    resp.GetState(),
		Path:     "/api/v1/auth/oauth",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, resp.GetAuthUrl(), http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	if !domain.ValidOAuthProviders[providerName] {
		h.redirectError(w, r, "unknown_provider")
		return
	}

	if providerErr := r.URL.Query().Get("error"); providerErr != "" {
		slog.Warn("oauth provider returned error", "provider", providerName, "error", providerErr)
		h.clearStateCookie(w)
		h.redirectError(w, r, "denied")
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		h.clearStateCookie(w)
		h.redirectError(w, r, "invalid_request")
		return
	}

	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || cookie.Value == "" || cookie.Value != state {
		h.clearStateCookie(w)
		h.redirectError(w, r, "invalid_state")
		return
	}
	h.clearStateCookie(w)

	resp, err := h.client.OAuthCallback(r.Context(), &pb.OAuthCallbackRequest{
		Provider: providerName,
		Code:     code,
		State:    state,
	})
	if err != nil {
		domainErr := grpcutil.GRPCToDomainError(err)
		slog.Error("oauth callback failed", "provider", providerName, "error", domainErr)
		h.redirectError(w, r, mapOAuthError(domainErr))
		return
	}

	expiresAt := time.Unix(resp.GetExpiresAt(), 0)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    resp.GetSessionId(),
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	middleware.SetCSRFCookie(w, expiresAt, h.cookieSecure)

	http.Redirect(w, r, h.frontendURL+"/files", http.StatusSeeOther)
}

func (h *OAuthHandler) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/api/v1/auth/oauth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *OAuthHandler) redirectError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, h.frontendURL+"/login?oauth_error="+code, http.StatusSeeOther)
}

func mapOAuthError(err error) string {
	switch {
	case err == nil:
		return "internal"
	case errors.Is(err, domain.ErrConflict):
		return "email_taken"
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrSessionExpired):
		return "denied"
	case errors.Is(err, domain.ErrInvalidInput):
		return "invalid_request"
	case errors.Is(err, domain.ErrForbidden):
		return "denied"
	default:
		return "internal"
	}
}
