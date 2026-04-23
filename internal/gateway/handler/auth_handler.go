package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/dto"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

type AuthHandler struct {
	client       pb.AuthServiceClient
	cookieSecure bool
}

func NewAuthHandler(client pb.AuthServiceClient, cookieSecure bool) *AuthHandler {
	return &AuthHandler{client: client, cookieSecure: cookieSecure}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.Me)
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.client.Register(r.Context(), &pb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, protoUserToDTO(resp.GetUser()))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.client.Login(r.Context(), &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
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
	httputil.JSON(w, http.StatusOK, protoUserToDTO(resp.GetUser()))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		httputil.JSON(w, http.StatusOK, nil)
		return
	}

	if _, err := h.client.Logout(r.Context(), &pb.LogoutRequest{SessionId: cookie.Value}); err != nil {
		slog.Error("logout failed", "error", grpcutil.GRPCToDomainError(err))
	}
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

	resp, err := h.client.ValidateSession(r.Context(), &pb.ValidateSessionRequest{
		SessionId: cookie.Value,
	})
	if err != nil {
		clearSessionCookie(w)
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoUserToDTO(resp.GetUser()))
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

func protoUserToDTO(info *pb.UserInfo) dto.UserResponse {
	resp := dto.UserResponse{
		ID:               info.GetId(),
		Username:         info.GetUsername(),
		Email:            info.GetEmail(),
		AvatarURL:        info.GetAvatarUrl(),
		Status:           info.GetStatus(),
		Description:      info.GetDescription(),
		IsAdmin:          info.GetIsAdmin(),
		Plan:             info.GetPlan(),
		TotalTimeSeconds: info.GetTotalTimeSeconds(),
		CreatedAt:        time.Unix(info.GetCreatedAt(), 0),
		UpdatedAt:        time.Unix(info.GetUpdatedAt(), 0),
	}
	if info.GetLastActiveAt() != 0 {
		t := time.Unix(info.GetLastActiveAt(), 0)
		resp.LastActiveAt = &t
	}
	return resp
}
