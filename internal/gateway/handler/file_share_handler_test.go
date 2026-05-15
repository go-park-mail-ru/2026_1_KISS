package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

func TestFileHandler_ShareByIdentifier_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().GetUserByIdentifier(gomock.Any(), gomock.Any()).Return(&pbauth.UserResponse{
		User: &pbauth.UserInfo{Id: 42, Email: "a@b.com"},
	}, nil)
	storageClient.EXPECT().ShareFile(gomock.Any(), gomock.Any()).Return(&pb.ShareFileResponse{
		Share: &pb.FileShareInfo{FileId: "file-1", UserId: 42, PermissionLevel: "download", CreatedAt: 1000},
	}, nil)

	h := NewFileHandler(storageClient, authClient, 0, "")
	body, _ := json.Marshal(map[string]string{"identifier": "a@b.com", "level": "download"})
	req := httptest.NewRequest("POST", "/api/v1/files/file-1/share", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "file-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ShareByIdentifier(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFileHandler_ShareByIdentifier_MissingIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	h := NewFileHandler(storageClient, authClient, 0, "")
	body, _ := json.Marshal(map[string]string{"identifier": "", "level": "view"})
	req := httptest.NewRequest("POST", "/api/v1/files/file-1/share", bytes.NewReader(body))
	req.SetPathValue("id", "file-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ShareByIdentifier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestFileHandler_ListShares_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().ListShares(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "forbidden"))

	h := NewFileHandler(storageClient, nil, 0, "")
	req := httptest.NewRequest("GET", "/api/v1/files/file-1/shares", nil)
	req.SetPathValue("id", "file-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListShares(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestFileHandler_RevokeShare_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().RevokeShare(gomock.Any(), gomock.Any()).Return(&pb.RevokeShareResponse{}, nil)

	h := NewFileHandler(storageClient, nil, 0, "")
	req := httptest.NewRequest("DELETE", "/api/v1/files/file-1/shares/2", nil)
	req.SetPathValue("id", "file-1")
	req.SetPathValue("userID", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.RevokeShare(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestFileHandler_SetPublic_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().SetFilePublic(gomock.Any(), gomock.Any()).Return(&pb.FileResponse{
		File: &pb.FileInfo{Id: "file-1", OwnerId: 1, IsPublic: true, ShareToken: "tok-1"},
	}, nil)

	h := NewFileHandler(storageClient, nil, 0, "")
	body, _ := json.Marshal(map[string]any{"is_public": true})
	req := httptest.NewRequest("PUT", "/api/v1/files/file-1/public", bytes.NewReader(body))
	req.SetPathValue("id", "file-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.SetPublic(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("share_token")) {
		t.Errorf("expected share_token in response, got %s", rec.Body.String())
	}
}

func TestFileHandler_SetPublic_InvalidExpiresAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	h := NewFileHandler(storageClient, nil, 0, "")
	body := strings.NewReader(`{"is_public": true, "expires_at": "not-a-date"}`)
	req := httptest.NewRequest("PUT", "/api/v1/files/file-1/public", body)
	req.SetPathValue("id", "file-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.SetPublic(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestFileHandler_Rename_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().RenameFile(gomock.Any(), gomock.Any()).Return(&pb.FileResponse{
		File: &pb.FileInfo{Id: "file-1", OwnerId: 1, Filename: "new.txt"},
	}, nil)

	h := NewFileHandler(storageClient, nil, 0, "")
	body, _ := json.Marshal(map[string]string{"filename": "new.txt"})
	req := httptest.NewRequest("PATCH", "/api/v1/files/file-1/rename", bytes.NewReader(body))
	req.SetPathValue("id", "file-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Rename(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestFileHandler_ListSharedWithMe_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().ListSharedWithMe(gomock.Any(), gomock.Any()).Return(&pb.ListFilesResponse{
		Files: []*pb.FileInfo{{Id: "f-1", OwnerId: 99, Filename: "doc.txt", YourPermission: "download"}},
		Total: 1,
	}, nil)

	h := NewFileHandler(storageClient, nil, 0, "")
	req := httptest.NewRequest("GET", "/api/v1/files/shared", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListSharedWithMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestFileHandler_DownloadShared_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().GetSharedFileByToken(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "expired"))

	h := NewFileHandler(storageClient, nil, 0, "")
	req := httptest.NewRequest("GET", "/api/v1/shared/files/tok-1", nil)
	req.SetPathValue("token", "tok-1")
	rec := httptest.NewRecorder()

	h.DownloadShared(rec, req)

	if rec.Code != http.StatusGone {
		t.Errorf("want 410, got %d", rec.Code)
	}
}

func TestFileHandler_DownloadShared_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().GetSharedFileByToken(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewFileHandler(storageClient, nil, 0, "")
	req := httptest.NewRequest("GET", "/api/v1/shared/files/tok-1", nil)
	req.SetPathValue("token", "tok-1")
	rec := httptest.NewRecorder()

	h.DownloadShared(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestFileHandler_Download_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageClient := mocks.NewMockStorageServiceClient(ctrl)

	storageClient.EXPECT().GetDownloadable(gomock.Any(), gomock.Any()).Return(&pb.GetDownloadableResponse{
		File: &pb.FileInfo{Id: "file-1"}, Allowed: false,
	}, nil)

	h := NewFileHandler(storageClient, nil, 0, "")
	req := httptest.NewRequest("GET", "/api/v1/files/file-1/download", nil)
	req.SetPathValue("id", "file-1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Download(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}
