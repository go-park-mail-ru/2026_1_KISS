package http

import (
	"context"
	"io"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type profileUsecase interface {
	UploadAvatar(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, contentType string) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, username, status, description string) (*domain.User, error)
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error
	ChangeEmail(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error)
}

// ProfileHandler handles profile-related HTTP endpoints.
type ProfileHandler struct {
	usecase     profileUsecase
	maxFileSize int64
}

// New creates a new ProfileHandler.
func New(uc profileUsecase, maxFileSize int64) *ProfileHandler {
	return &ProfileHandler{usecase: uc, maxFileSize: maxFileSize}
}

// RegisterRoutes registers profile endpoints on the given mux.
func (h *ProfileHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("POST /api/v1/users/me/avatar", authMw(http.HandlerFunc(h.UploadAvatar)))
	mux.Handle("PUT /api/v1/users/me", authMw(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("PUT /api/v1/users/me/password", authMw(http.HandlerFunc(h.ChangePassword)))
	mux.Handle("PUT /api/v1/users/me/email", authMw(http.HandlerFunc(h.ChangeEmail)))
}

// UploadAvatar handles multipart/form-data avatar uploads.
func (h *ProfileHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxFileSize+1024)

	if err := r.ParseMultipartForm(h.maxFileSize); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid file upload")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	updated, err := h.usecase.UploadAvatar(r.Context(), user.ID, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, NewUserResponse(updated))
}

// UpdateProfile handles profile field updates.
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req UpdateProfileRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	updated, err := h.usecase.UpdateProfile(r.Context(), user.ID, req.Username, req.Status, req.Description)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, NewUserResponse(updated))
}

// ChangePassword handles password changes.
func (h *ProfileHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.usecase.ChangePassword(r.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

// ChangeEmail handles email changes.
func (h *ProfileHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	var req ChangeEmailRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	updated, err := h.usecase.ChangeEmail(r.Context(), user.ID, req.NewEmail, req.Password)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, NewUserResponse(updated))
}

func mapDomainError(w http.ResponseWriter, err error) {
	httputil.MapDomainError(w, err)
}
