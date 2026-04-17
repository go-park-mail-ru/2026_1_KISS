package handler

import (
	"io"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

type ProfileHandler struct {
	client      pb.AuthServiceClient
	maxFileSize int64
}

func NewProfileHandler(client pb.AuthServiceClient, maxFileSize int64) *ProfileHandler {
	return &ProfileHandler{client: client, maxFileSize: maxFileSize}
}

func (h *ProfileHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("POST /api/v1/users/me/avatar", authMw(http.HandlerFunc(h.UploadAvatar)))
	mux.Handle("PUT /api/v1/users/me", authMw(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("PUT /api/v1/users/me/password", authMw(http.HandlerFunc(h.ChangePassword)))
	mux.Handle("PUT /api/v1/users/me/email", authMw(http.HandlerFunc(h.ChangeEmail)))
}

type updateProfileRequest struct {
	Username    string `json:"username"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type changeEmailRequest struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}

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

	data, err := io.ReadAll(file)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "failed to read file")
		return
	}

	stream, err := h.client.UploadAvatar(r.Context())
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	chunkSize := 64 * 1024
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := &pb.UploadAvatarChunk{Data: data[i:end]}
		if i == 0 {
			chunk.UserId = user.ID
			chunk.Filename = header.Filename
			chunk.FileSize = header.Size
		}
		if err := stream.Send(chunk); err != nil {
			httputil.Error(w, http.StatusInternalServerError, "upload failed")
			return
		}
	}

	if len(data) == 0 {
		if err := stream.Send(&pb.UploadAvatarChunk{
			UserId:   user.ID,
			Filename: header.Filename,
			FileSize: header.Size,
			Data:     []byte{},
		}); err != nil {
			httputil.Error(w, http.StatusInternalServerError, "upload failed")
			return
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoUserToDTO(resp.GetUser()))
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req updateProfileRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.client.UpdateProfile(r.Context(), &pb.UpdateProfileRequest{
		UserId:      user.ID,
		Username:    req.Username,
		Status:      req.Status,
		Description: req.Description,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoUserToDTO(resp.GetUser()))
}

func (h *ProfileHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	_, err := h.client.ChangePassword(r.Context(), &pb.ChangePasswordRequest{
		UserId:          user.ID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

func (h *ProfileHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	var req changeEmailRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.client.ChangeEmail(r.Context(), &pb.ChangeEmailRequest{
		UserId:   user.ID,
		NewEmail: req.NewEmail,
		Password: req.Password,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoUserToDTO(resp.GetUser()))
}

