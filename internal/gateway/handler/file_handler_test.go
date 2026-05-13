package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

func withUserPlan(r *http.Request, userID int64, plan string) *http.Request {
	user := &domain.User{ID: userID, Plan: plan}
	ctx := middleware.SetUserInContext(r.Context(), user)
	return r.WithContext(ctx)
}

func TestFileHandler_List_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().ListFiles(gomock.Any(), gomock.Any()).Return(&pb.ListFilesResponse{
		Files: []*pb.FileInfo{
			{Id: "f-1", OwnerId: 1, Category: "datasets", Filename: "data.csv", Url: "/uploads/datasets/uuid.csv", MimeType: "text/csv", Size: 100, CreatedAt: 1000},
		},
		Total: 1,
	}, nil)

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("GET", "/api/v1/files?category=datasets&limit=10", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestFileHandler_List_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("GET", "/api/v1/files", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestFileHandler_Get_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().GetFile(gomock.Any(), gomock.Any()).Return(&pb.FileResponse{
		File: &pb.FileInfo{Id: "f-1", OwnerId: 1, Category: "files", Filename: "readme.txt"},
	}, nil)

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("GET", "/api/v1/files/f-1", nil)
	req.SetPathValue("id", "f-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestFileHandler_Get_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().GetFile(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("GET", "/api/v1/files/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestFileHandler_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).Return(&pb.DeleteFileResponse{}, nil)

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("DELETE", "/api/v1/files/f-1", nil)
	req.SetPathValue("id", "f-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestFileHandler_Delete_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("DELETE", "/api/v1/files/f-1", nil)
	req.SetPathValue("id", "f-1")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestFileHandler_Upload_NoFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	h := NewFileHandler(storageClient, 10*1024*1024)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestFileHandler_Upload_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	h := NewFileHandler(storageClient, 10*1024*1024)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "test.txt")
	_, _ = part.Write([]byte("hello"))
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestFileHandler_Usage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().GetUserStorageStats(gomock.Any(), gomock.Any()).
		Return(&pb.GetStorageStatsResponse{TotalSizeBytes: 1024, TotalFiles: 3}, nil)

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("GET", "/api/v1/files/usage", nil)
	req = withUserPlan(req, 1, domain.PlanPro)
	rec := httptest.NewRecorder()

	h.Usage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	body := envelope.Data
	if body["plan"] != domain.PlanPro {
		t.Errorf("plan: want %q, got %v", domain.PlanPro, body["plan"])
	}
	if used, _ := body["used"].(float64); used != 1024 {
		t.Errorf("used: want 1024, got %v", body["used"])
	}
}

func TestFileHandler_Upload_QuotaExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().GetUserStorageStats(gomock.Any(), gomock.Any()).
		Return(&pb.GetStorageStatsResponse{TotalSizeBytes: 128 * 1024 * 1024, TotalFiles: 1}, nil)

	h := NewFileHandler(storageClient, 200*1024*1024)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "big.bin")
	_, _ = part.Write(bytes.Repeat([]byte("x"), 1024))
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = withUserPlan(req, 1, domain.PlanFree)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusInsufficientStorage {
		t.Errorf("want 507, got %d", rec.Code)
	}
}

func TestFileHandler_List_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().ListFiles(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "internal"))

	h := NewFileHandler(storageClient, 10*1024*1024)
	req := httptest.NewRequest("GET", "/api/v1/files", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}
