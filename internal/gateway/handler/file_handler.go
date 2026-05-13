package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/quota"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type FileHandler struct {
	storageClient pb.StorageServiceClient
	maxFileSize   int64
}

func NewFileHandler(storageClient pb.StorageServiceClient, maxFileSize int64) *FileHandler {
	return &FileHandler{storageClient: storageClient, maxFileSize: maxFileSize}
}

func (h *FileHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("POST /api/v1/files/upload", authMw(http.HandlerFunc(h.Upload)))
	mux.Handle("GET /api/v1/files", authMw(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/files/usage", authMw(http.HandlerFunc(h.Usage)))
	mux.Handle("GET /api/v1/files/{id}", authMw(http.HandlerFunc(h.Get)))
	mux.Handle("DELETE /api/v1/files/{id}", authMw(http.HandlerFunc(h.Delete)))
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

type fileResponse struct {
	ID         string    `json:"id"`
	OwnerID    int64     `json:"owner_id"`
	NotebookID *int64    `json:"notebook_id,omitempty"`
	Category   string    `json:"category"`
	Filename   string    `json:"filename"`
	URL        string    `json:"url"`
	MIMEType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
}

func fileInfoToResponse(f *pb.FileInfo) fileResponse {
	resp := fileResponse{
		ID:       f.GetId(),
		OwnerID:  f.GetOwnerId(),
		Category: f.GetCategory(),
		Filename: f.GetFilename(),
		URL:      f.GetUrl(),
		MIMEType: f.GetMimeType(),
		Size:     f.GetSize(),
	}
	if f.GetCreatedAt() != 0 {
		resp.CreatedAt = time.Unix(f.GetCreatedAt(), 0)
	}
	if f.NotebookId != nil {
		nbID := f.GetNotebookId()
		resp.NotebookID = &nbID
	}
	return resp
}
