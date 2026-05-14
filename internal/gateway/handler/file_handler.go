package handler

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/quota"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type FileHandler struct {
	storageClient pb.StorageServiceClient
	authClient    pbauth.AuthServiceClient
	maxFileSize   int64
	uploadDir     string
}

func NewFileHandler(storageClient pb.StorageServiceClient, authClient pbauth.AuthServiceClient, maxFileSize int64, uploadDir string) *FileHandler {
	return &FileHandler{
		storageClient: storageClient,
		authClient:    authClient,
		maxFileSize:   maxFileSize,
		uploadDir:     uploadDir,
	}
}

func (h *FileHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("POST /api/v1/files/upload", authMw(http.HandlerFunc(h.Upload)))
	mux.Handle("GET /api/v1/files", authMw(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/files/usage", authMw(http.HandlerFunc(h.Usage)))
	mux.Handle("GET /api/v1/files/shared", authMw(http.HandlerFunc(h.ListSharedWithMe)))
	mux.Handle("GET /api/v1/files/{id}", authMw(http.HandlerFunc(h.Get)))
	mux.Handle("DELETE /api/v1/files/{id}", authMw(http.HandlerFunc(h.Delete)))
	mux.Handle("GET /api/v1/files/{id}/download", authMw(http.HandlerFunc(h.Download)))
	mux.Handle("POST /api/v1/files/{id}/share", authMw(http.HandlerFunc(h.ShareByIdentifier)))
	mux.Handle("GET /api/v1/files/{id}/shares", authMw(http.HandlerFunc(h.ListShares)))
	mux.Handle("PUT /api/v1/files/{id}/shares/{userID}", authMw(http.HandlerFunc(h.UpdateShare)))
	mux.Handle("DELETE /api/v1/files/{id}/shares/{userID}", authMw(http.HandlerFunc(h.RevokeShare)))
	mux.Handle("PUT /api/v1/files/{id}/public", authMw(http.HandlerFunc(h.SetPublic)))
	mux.Handle("PATCH /api/v1/files/{id}/rename", authMw(http.HandlerFunc(h.Rename)))
	mux.Handle("GET /api/v1/shared/files/{token}", http.HandlerFunc(h.DownloadShared))
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxFileSize+1024)

	if err := r.ParseMultipartForm(h.maxFileSize); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid file upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	category := r.FormValue("category")
	if category == "" {
		category = "files"
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := quota.LimitFor(user.Plan)
	if !quota.IsUnlimited(user.Plan) {
		stats, statsErr := h.storageClient.GetUserStorageStats(r.Context(), &pb.GetUserStorageStatsRequest{UserId: user.ID})
		if statsErr != nil {
			httputil.MapDomainError(w, grpcutil.GRPCToDomainError(statsErr))
			return
		}
		if stats.GetTotalSizeBytes()+header.Size > limit {
			httputil.Error(w, http.StatusInsufficientStorage, "storage quota exceeded")
			return
		}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "failed to read file")
		return
	}

	stream, err := h.storageClient.UploadFile(r.Context())
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	chunkSize := 64 * 1024
	for i := 0; i < len(data); i += chunkSize {
		end := min(i+chunkSize, len(data))
		chunk := &pb.UploadFileChunk{Data: data[i:end]}
		if i == 0 {
			chunk.OwnerId = user.ID
			chunk.Filename = header.Filename
			chunk.FileSize = header.Size
			chunk.Category = category
		}
		if err := stream.Send(chunk); err != nil {
			httputil.Error(w, http.StatusInternalServerError, "upload failed")
			return
		}
	}

	if len(data) == 0 {
		if err := stream.Send(&pb.UploadFileChunk{
			OwnerId:  user.ID,
			Filename: header.Filename,
			FileSize: header.Size,
			Category: category,
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

	httputil.JSON(w, http.StatusCreated, fileInfoToResponse(resp.GetFile()))
}

func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	category := r.URL.Query().Get("category")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.storageClient.ListFiles(r.Context(), &pb.ListFilesRequest{
		UserId:   user.ID,
		Category: category,
		Limit:    int32(limit),  //nolint:gosec
		Offset:   int32(offset), //nolint:gosec
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	files := make([]fileResponse, len(resp.GetFiles()))
	for i, f := range resp.GetFiles() {
		files[i] = fileInfoToResponse(f)
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"files": files,
		"total": resp.GetTotal(),
	})
}

func (h *FileHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileID := r.PathValue("id")
	resp, err := h.storageClient.GetFile(r.Context(), &pb.GetFileRequest{
		FileId: fileID,
		UserId: user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, fileInfoToResponse(resp.GetFile()))
}

func (h *FileHandler) Usage(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	stats, err := h.storageClient.GetUserStorageStats(r.Context(), &pb.GetUserStorageStatsRequest{UserId: user.ID})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	limit := quota.LimitFor(user.Plan)
	unlimited := quota.IsUnlimited(user.Plan)

	resp := map[string]any{
		"used":        stats.GetTotalSizeBytes(),
		"limit":       limit,
		"unlimited":   unlimited,
		"plan":        user.Plan,
		"files_count": stats.GetTotalFiles(),
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileID := r.PathValue("id")
	_, err := h.storageClient.DeleteFile(r.Context(), &pb.DeleteFileRequest{
		FileId: fileID,
		UserId: user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileID := r.PathValue("id")
	resp, err := h.storageClient.GetDownloadable(r.Context(), &pb.GetDownloadableRequest{
		FileId: fileID,
		UserId: user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	if !resp.GetAllowed() {
		httputil.Error(w, http.StatusForbidden, "no download permission")
		return
	}

	h.serveAndCount(w, r, resp.GetFile())
}

func (h *FileHandler) DownloadShared(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		httputil.Error(w, http.StatusNotFound, "share not found")
		return
	}

	resp, err := h.storageClient.GetSharedFileByToken(r.Context(), &pb.GetSharedFileByTokenRequest{Token: token})
	if err != nil {
		domainErr := grpcutil.GRPCToDomainError(err)
		if errors.Is(domainErr, domain.ErrForbidden) {
			httputil.Error(w, http.StatusGone, "share link expired")
			return
		}
		if errors.Is(domainErr, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "share not found")
			return
		}
		httputil.MapDomainError(w, domainErr)
		return
	}

	h.serveAndCount(w, r, resp.GetFile())
}

func (h *FileHandler) serveAndCount(w http.ResponseWriter, r *http.Request, info *pb.FileInfo) {
	if info == nil {
		httputil.Error(w, http.StatusNotFound, "file not found")
		return
	}
	storageKey := info.GetStorageKey()
	if storageKey == "" {
		httputil.Error(w, http.StatusInternalServerError, "storage key missing")
		return
	}
	if strings.Contains(storageKey, "..") {
		httputil.Error(w, http.StatusBadRequest, "invalid storage key")
		return
	}
	fullPath := filepath.Join(h.uploadDir, filepath.Clean("/"+storageKey))

	w.Header().Set("Content-Disposition", `attachment; filename="`+sanitizeContentDisposition(info.GetFilename())+`"`)
	if info.GetMimeType() != "" {
		w.Header().Set("Content-Type", info.GetMimeType())
	}
	http.ServeFile(w, r, fullPath)

	_, _ = h.storageClient.IncrementDownloadCount(r.Context(), &pb.IncrementDownloadCountRequest{FileId: info.GetId()})
}

func sanitizeContentDisposition(name string) string {
	r := strings.NewReplacer(`"`, ``, "\r", "", "\n", "", "\\", "")
	return r.Replace(name)
}

type shareByIdentifierRequest struct {
	Identifier string `json:"identifier"`
	Level      string `json:"level"`
}

type updateShareRequest struct {
	Level string `json:"level"`
}

type setFilePublicRequest struct {
	IsPublic  bool    `json:"is_public"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

type renameRequest struct {
	Filename string `json:"filename"`
}

type shareResponse struct {
	FileID    string    `json:"file_id"`
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	Level     string    `json:"permission_level"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *FileHandler) ShareByIdentifier(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req shareByIdentifierRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Identifier == "" || req.Level == "" {
		httputil.Error(w, http.StatusBadRequest, "identifier and level are required")
		return
	}

	authResp, err := h.authClient.GetUserByIdentifier(r.Context(), &pbauth.GetUserByIdentifierRequest{Identifier: req.Identifier})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	target := authResp.GetUser()

	fileID := r.PathValue("id")
	resp, err := h.storageClient.ShareFile(r.Context(), &pb.ShareFileRequest{
		RequesterId:     user.ID,
		FileId:          fileID,
		TargetUserId:    target.GetId(),
		PermissionLevel: req.Level,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	share := resp.GetShare()
	out := shareResponse{
		FileID: share.GetFileId(),
		UserID: share.GetUserId(),
		Email:  target.GetEmail(),
		Level:  share.GetPermissionLevel(),
	}
	if share.GetCreatedAt() > 0 {
		out.CreatedAt = time.Unix(share.GetCreatedAt(), 0)
	}
	httputil.JSON(w, http.StatusOK, out)
}

func (h *FileHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileID := r.PathValue("id")
	resp, err := h.storageClient.ListShares(r.Context(), &pb.ListSharesRequest{RequesterId: user.ID, FileId: fileID})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	shares := make([]shareResponse, len(resp.GetShares()))
	for i, sh := range resp.GetShares() {
		shares[i] = shareResponse{
			FileID: sh.GetFileId(),
			UserID: sh.GetUserId(),
			Email:  sh.GetEmail(),
			Level:  sh.GetPermissionLevel(),
		}
		if sh.GetCreatedAt() > 0 {
			shares[i].CreatedAt = time.Unix(sh.GetCreatedAt(), 0)
		}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"shares": shares})
}

func (h *FileHandler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	targetUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateShareRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fileID := r.PathValue("id")
	resp, err := h.storageClient.ShareFile(r.Context(), &pb.ShareFileRequest{
		RequesterId:     user.ID,
		FileId:          fileID,
		TargetUserId:    targetUserID,
		PermissionLevel: req.Level,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	share := resp.GetShare()
	out := shareResponse{
		FileID: share.GetFileId(),
		UserID: share.GetUserId(),
		Level:  share.GetPermissionLevel(),
	}
	if share.GetCreatedAt() > 0 {
		out.CreatedAt = time.Unix(share.GetCreatedAt(), 0)
	}
	httputil.JSON(w, http.StatusOK, out)
}

func (h *FileHandler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	targetUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	fileID := r.PathValue("id")
	_, err = h.storageClient.RevokeShare(r.Context(), &pb.RevokeShareRequest{
		RequesterId:  user.ID,
		FileId:       fileID,
		TargetUserId: targetUserID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, nil)
}

func (h *FileHandler) SetPublic(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setFilePublicRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var expiresAt int64
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			httputil.Error(w, http.StatusBadRequest, "invalid expires_at format")
			return
		}
		expiresAt = t.Unix()
	}

	fileID := r.PathValue("id")
	resp, err := h.storageClient.SetFilePublic(r.Context(), &pb.SetFilePublicRequest{
		RequesterId: user.ID,
		FileId:      fileID,
		IsPublic:    req.IsPublic,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, fileInfoToResponse(resp.GetFile()))
}

func (h *FileHandler) Rename(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req renameRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fileID := r.PathValue("id")
	resp, err := h.storageClient.RenameFile(r.Context(), &pb.RenameFileRequest{
		RequesterId: user.ID,
		FileId:      fileID,
		Filename:    req.Filename,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, fileInfoToResponse(resp.GetFile()))
}

func (h *FileHandler) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.storageClient.ListSharedWithMe(r.Context(), &pb.ListSharedWithMeRequest{
		UserId: user.ID,
		Limit:  int32(limit),  //nolint:gosec
		Offset: int32(offset), //nolint:gosec
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	files := make([]fileResponse, len(resp.GetFiles()))
	for i, f := range resp.GetFiles() {
		files[i] = fileInfoToResponse(f)
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"files": files,
		"total": resp.GetTotal(),
	})
}

type fileResponse struct {
	ID             string     `json:"id"`
	OwnerID        int64      `json:"owner_id"`
	NotebookID     *int64     `json:"notebook_id,omitempty"`
	Category       string     `json:"category"`
	Filename       string     `json:"filename"`
	URL            string     `json:"url"`
	MIMEType       string     `json:"mime_type"`
	Size           int64      `json:"size"`
	CreatedAt      time.Time  `json:"created_at"`
	IsPublic       bool       `json:"is_public"`
	ShareToken     *string    `json:"share_token,omitempty"`
	ShareExpiresAt *time.Time `json:"share_expires_at,omitempty"`
	DownloadsCount int64      `json:"downloads_count"`
	YourPermission string     `json:"your_permission,omitempty"`
	DownloadURL    string     `json:"download_url"`
	PublicURL      *string    `json:"public_url,omitempty"`
}

func fileInfoToResponse(f *pb.FileInfo) fileResponse {
	resp := fileResponse{
		ID:             f.GetId(),
		OwnerID:        f.GetOwnerId(),
		Category:       f.GetCategory(),
		Filename:       f.GetFilename(),
		URL:            f.GetUrl(),
		MIMEType:       f.GetMimeType(),
		Size:           f.GetSize(),
		IsPublic:       f.GetIsPublic(),
		DownloadsCount: f.GetDownloadsCount(),
		YourPermission: f.GetYourPermission(),
		DownloadURL:    "/api/v1/files/" + f.GetId() + "/download",
	}
	if f.GetCreatedAt() != 0 {
		resp.CreatedAt = time.Unix(f.GetCreatedAt(), 0)
	}
	if f.NotebookId != nil {
		nbID := f.GetNotebookId()
		resp.NotebookID = &nbID
	}
	if f.GetShareToken() != "" {
		token := f.GetShareToken()
		resp.ShareToken = &token
		publicURL := "/api/v1/shared/files/" + token
		resp.PublicURL = &publicURL
	}
	if f.GetShareExpiresAt() > 0 {
		t := time.Unix(f.GetShareExpiresAt(), 0)
		resp.ShareExpiresAt = &t
	}
	return resp
}
