package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
)

func newTestUsecase(ctrl *gomock.Controller) (*StorageUsecase, *mocks.MockFileRepository, *mocks.MockFileStorage) {
	uc, repo, fs, _ := newTestUsecaseWithShare(ctrl)
	return uc, repo, fs
}

func newTestUsecaseWithShare(ctrl *gomock.Controller) (*StorageUsecase, *mocks.MockFileRepository, *mocks.MockFileStorage, *mocks.MockFileShareRepository) {
	repo := mocks.NewMockFileRepository(ctrl)
	shareRepo := mocks.NewMockFileShareRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)
	maxSizes := map[domain.FileCategory]int64{
		domain.FileCategoryGeneral: 10 * 1024 * 1024,
		domain.FileCategoryDataset: 50 * 1024 * 1024,
		domain.FileCategoryAvatar:  2 * 1024 * 1024,
	}
	uc := New(repo, shareRepo, fs, maxSizes)
	return uc, repo, fs, shareRepo
}

func TestUploadFile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, fs := newTestUsecase(ctrl)

	data := strings.NewReader("hello world csv data")
	fs.EXPECT().Save(gomock.Any(), gomock.Any()).Return("/uploads/datasets/uuid.csv", nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f *domain.File) error {
		f.ID = "file-1"
		f.CreatedAt = time.Now()
		return nil
	})

	file, err := uc.UploadFile(context.Background(), 1, domain.FileCategoryDataset, "test.csv", data, 20, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.ID != "file-1" {
		t.Errorf("expected file-1, got %s", file.ID)
	}
	if file.Category != domain.FileCategoryDataset {
		t.Errorf("expected datasets, got %s", file.Category)
	}
}

func TestUploadFile_InvalidCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, _, _ := newTestUsecase(ctrl)

	_, err := uc.UploadFile(context.Background(), 1, "invalid", "test.txt", strings.NewReader("data"), 4, nil)
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestUploadFile_FileTooLarge(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, _, _ := newTestUsecase(ctrl)

	_, err := uc.UploadFile(context.Background(), 1, domain.FileCategoryAvatar, "big.jpg", strings.NewReader("data"), 3*1024*1024, nil)
	if err == nil {
		t.Fatal("expected error for file too large")
	}
}

func TestGetFile_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, shareRepo := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{
		ID:      "file-1",
		OwnerID: 99,
	}, nil)
	shareRepo.EXPECT().GetPermission(gomock.Any(), "file-1", int64(1)).Return(nil, domain.ErrNotFound)

	_, err := uc.GetFile(context.Background(), "file-1", 1)
	if err != domain.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestGetFile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{
		ID:      "file-1",
		OwnerID: 1,
	}, nil)

	file, err := uc.GetFile(context.Background(), "file-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.ID != "file-1" {
		t.Errorf("expected file-1, got %s", file.ID)
	}
}

func TestDeleteFile_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{
		ID:      "file-1",
		OwnerID: 99,
	}, nil)

	err := uc.DeleteFile(context.Background(), "file-1", 1)
	if err != domain.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteFile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, fs := newTestUsecase(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{
		ID:      "file-1",
		OwnerID: 1,
		URL:     "/uploads/files/uuid.txt",
	}, nil)
	repo.EXPECT().Delete(gomock.Any(), "file-1").Return(nil)
	fs.EXPECT().Delete("/uploads/files/uuid.txt").Return(nil)

	err := uc.DeleteFile(context.Background(), "file-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListFiles_DefaultLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	repo.EXPECT().ListByOwner(gomock.Any(), int64(1), "", 20, 0).Return(nil, 0, nil)

	_, _, err := uc.ListFiles(context.Background(), 1, "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminListFiles(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	repo.EXPECT().ListAll(gomock.Any(), "datasets", int64(0), 20, 0).Return([]domain.File{
		{ID: "f-1", Category: domain.FileCategoryDataset},
	}, 1, nil)

	files, total, err := uc.AdminListFiles(context.Background(), "datasets", 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1, got %d", total)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestListFiles_CapLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	repo.EXPECT().ListByOwner(gomock.Any(), int64(1), "", 100, 0).Return(nil, 0, nil)

	_, _, err := uc.ListFiles(context.Background(), 1, "", 999, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMimeToExt(t *testing.T) {
	tests := []struct {
		mime string
		ext  string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/bmp", ".bmp"},
		{"application/pdf", ".pdf"},
		{"text/csv", ".csv"},
		{"application/json", ".json"},
		{"text/plain", ".txt"},
		{"application/octet-stream", ".bin"},
	}
	for _, tt := range tests {
		got := mimeToExt(tt.mime)
		if got != tt.ext {
			t.Errorf("mimeToExt(%s) = %s, want %s", tt.mime, got, tt.ext)
		}
	}
}

func TestGetStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	expected := &domain.StorageStats{
		TotalFiles:     10,
		TotalSizeBytes: 2048,
	}
	repo.EXPECT().GetStats(gomock.Any()).Return(expected, nil)

	stats, err := uc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalFiles != 10 {
		t.Errorf("expected 10, got %d", stats.TotalFiles)
	}
}

func TestGetUserStorageStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	expected := &domain.StorageStats{
		TotalFiles:      3,
		TotalSizeBytes:  1024,
		FilesByCategory: map[domain.FileCategory]int64{"datasets": 2, "files": 1},
		SizeByCategory:  map[domain.FileCategory]int64{"datasets": 800, "files": 224},
	}

	repo.EXPECT().GetStatsByOwner(gomock.Any(), int64(1)).Return(expected, nil)

	stats, err := uc.GetUserStorageStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalFiles != 3 {
		t.Errorf("expected 3 files, got %d", stats.TotalFiles)
	}
	if stats.TotalSizeBytes != 1024 {
		t.Errorf("expected 1024 bytes, got %d", stats.TotalSizeBytes)
	}
}

func TestGetUserStorageStats_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _ := newTestUsecase(ctrl)

	repo.EXPECT().GetStatsByOwner(gomock.Any(), int64(99)).Return(nil, domain.ErrNotFound)

	_, err := uc.GetUserStorageStats(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
