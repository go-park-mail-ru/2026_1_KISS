package usecase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/filestorage"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/storage/repository"
)

type StorageUsecase struct {
	fileRepo    repository.FileRepository
	fileStorage filestorage.FileStorage
	maxSizes    map[domain.FileCategory]int64
}

func New(fileRepo repository.FileRepository, fs filestorage.FileStorage, maxSizes map[domain.FileCategory]int64) *StorageUsecase {
	return &StorageUsecase{
		fileRepo:    fileRepo,
		fileStorage: fs,
		maxSizes:    maxSizes,
	}
}

func (uc *StorageUsecase) UploadFile(ctx context.Context, ownerID int64, category domain.FileCategory, filename string, data io.Reader, fileSize int64, notebookID *int64) (*domain.File, error) {
	logger.Info(ctx, "usecase.storage.UploadFile", "owner_id", ownerID, "category", category, "filename", filename)

	if !domain.ValidFileCategory(category) {
		return nil, fmt.Errorf("%w: invalid category", domain.ErrInvalidInput)
	}

	maxSize, ok := uc.maxSizes[category]
	if !ok {
		maxSize = uc.maxSizes[domain.FileCategoryGeneral]
	}
	if fileSize > maxSize {
		return nil, fmt.Errorf("%w: file too large (max %d bytes)", domain.ErrInvalidInput, maxSize)
	}

	sniffBuf := make([]byte, 512)
	n, err := io.ReadFull(data, sniffBuf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("read file header: %w", err)
	}
	sniffBuf = sniffBuf[:n]
	mimeType := strings.Split(http.DetectContentType(sniffBuf), ";")[0]

	combined := io.MultiReader(
		strings.NewReader(string(sniffBuf)),
		data,
	)

	ext := filepath.Ext(filename)
	if ext == "" {
		ext = mimeToExt(mimeType)
	}
	storageKey := string(category) + "/" + uuid.New().String() + ext

	url, err := uc.fileStorage.Save(storageKey, combined)
	if err != nil {
		logger.Error(ctx, "usecase.storage.UploadFile", "error", err)
		return nil, fmt.Errorf("save file: %w", err)
	}

	file := &domain.File{
		OwnerID:    ownerID,
		NotebookID: notebookID,
		Category:   category,
		Filename:   filename,
		StorageKey: storageKey,
		URL:        url,
		MIMEType:   mimeType,
		Size:       fileSize,
	}

	if err := uc.fileRepo.Create(ctx, file); err != nil {
		_ = uc.fileStorage.Delete(url)
		return nil, err
	}

	logger.Info(ctx, "usecase.storage.UploadFile", "file_id", file.ID, "status", "ok")
	return file, nil
}

func (uc *StorageUsecase) GetFile(ctx context.Context, fileID string, userID int64) (*domain.File, error) {
	logger.Info(ctx, "usecase.storage.GetFile", "file_id", fileID, "user_id", userID)

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if file.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	return file, nil
}

func (uc *StorageUsecase) ListFiles(ctx context.Context, userID int64, category string, limit, offset int) ([]domain.File, int, error) {
	logger.Info(ctx, "usecase.storage.ListFiles", "user_id", userID, "category", category)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return uc.fileRepo.ListByOwner(ctx, userID, category, limit, offset)
}

func (uc *StorageUsecase) DeleteFile(ctx context.Context, fileID string, userID int64) error {
	logger.Info(ctx, "usecase.storage.DeleteFile", "file_id", fileID, "user_id", userID)

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	if file.OwnerID != userID {
		return domain.ErrForbidden
	}

	if err := uc.fileRepo.Delete(ctx, fileID); err != nil {
		return err
	}

	_ = uc.fileStorage.Delete(file.URL)
	logger.Info(ctx, "usecase.storage.DeleteFile", "file_id", fileID, "status", "ok")
	return nil
}

func (uc *StorageUsecase) DeleteFileByURL(ctx context.Context, url string) error {
	logger.Info(ctx, "usecase.storage.DeleteFileByURL", "url", url)

	if url == "" {
		return nil
	}

	storageKey, err := uc.fileRepo.DeleteByURL(ctx, url)
	if err != nil {
		return err
	}
	if storageKey != "" {
		_ = uc.fileStorage.Delete(url)
	}
	return nil
}

func (uc *StorageUsecase) GetStats(ctx context.Context) (*domain.StorageStats, error) {
	return uc.fileRepo.GetStats(ctx)
}

func (uc *StorageUsecase) AdminListFiles(ctx context.Context, category string, ownerID int64, limit, offset int) ([]domain.File, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return uc.fileRepo.ListAll(ctx, category, ownerID, limit, offset)
}

func mimeToExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/bmp":
		return ".bmp"
	case "application/pdf":
		return ".pdf"
	case "text/csv":
		return ".csv"
	case "application/json":
		return ".json"
	case "text/plain":
		return ".txt"
	default:
		return ".bin"
	}
}
