package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

const maxFilenameLen = 255

func (uc *StorageUsecase) computePermission(ctx context.Context, file *domain.File, userID int64) string {
	if file.OwnerID == userID && userID > 0 {
		return domain.FilePermissionOwner
	}
	if file.IsPublic && !shareExpired(file) {
		return domain.FilePermissionPublic
	}
	if userID > 0 && uc.shareRepo != nil {
		share, err := uc.shareRepo.GetPermission(ctx, file.ID, userID)
		if err == nil && share != nil {
			return share.Level
		}
	}
	return ""
}

func shareExpired(file *domain.File) bool {
	if file.ShareExpiresAt == nil {
		return false
	}
	return time.Now().After(*file.ShareExpiresAt)
}

func (uc *StorageUsecase) ShareFile(ctx context.Context, requesterID int64, fileID string, targetUserID int64, level string) (*domain.FileShare, error) {
	logger.Info(ctx, "usecase.storage.ShareFile", "file_id", fileID, "target_user_id", targetUserID, "level", level)

	if !domain.ValidFileShareLevel(level) {
		return nil, fmt.Errorf("%w: invalid permission level", domain.ErrInvalidInput)
	}
	if targetUserID <= 0 {
		return nil, fmt.Errorf("%w: invalid target user id", domain.ErrInvalidInput)
	}

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.OwnerID != requesterID {
		return nil, domain.ErrForbidden
	}
	if targetUserID == requesterID {
		return nil, fmt.Errorf("%w: cannot share with yourself", domain.ErrInvalidInput)
	}

	share := &domain.FileShare{FileID: fileID, UserID: targetUserID, Level: level}
	if err := uc.shareRepo.Upsert(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (uc *StorageUsecase) RevokeShare(ctx context.Context, requesterID int64, fileID string, targetUserID int64) error {
	logger.Info(ctx, "usecase.storage.RevokeShare", "file_id", fileID, "target_user_id", targetUserID)

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.OwnerID != requesterID {
		return domain.ErrForbidden
	}
	return uc.shareRepo.Delete(ctx, fileID, targetUserID)
}

func (uc *StorageUsecase) ListShares(ctx context.Context, requesterID int64, fileID string) ([]domain.FileShare, error) {
	logger.Info(ctx, "usecase.storage.ListShares", "file_id", fileID)

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.OwnerID != requesterID {
		return nil, domain.ErrForbidden
	}
	return uc.shareRepo.GetByFileID(ctx, fileID)
}

func (uc *StorageUsecase) ListSharedWithMe(ctx context.Context, userID int64, limit, offset int) ([]domain.File, int, error) {
	logger.Info(ctx, "usecase.storage.ListSharedWithMe", "user_id", userID)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return uc.shareRepo.ListByUserID(ctx, userID, limit, offset)
}

func (uc *StorageUsecase) SetFilePublic(ctx context.Context, requesterID int64, fileID string, isPublic bool, expiresAt *time.Time) (*domain.File, error) {
	logger.Info(ctx, "usecase.storage.SetFilePublic", "file_id", fileID, "is_public", isPublic)

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.OwnerID != requesterID {
		return nil, domain.ErrForbidden
	}

	var token *string
	var exp *time.Time
	if isPublic {
		if file.ShareToken != nil {
			token = file.ShareToken
		} else {
			t := uuid.New().String()
			token = &t
		}
		exp = expiresAt
	}

	if err := uc.fileRepo.SetPublic(ctx, fileID, requesterID, isPublic, token, exp); err != nil {
		return nil, err
	}
	file.IsPublic = isPublic
	file.ShareToken = token
	file.ShareExpiresAt = exp
	file.YourPermission = domain.FilePermissionOwner
	return file, nil
}

func (uc *StorageUsecase) GetSharedFileByToken(ctx context.Context, token string) (*domain.File, error) {
	logger.Info(ctx, "usecase.storage.GetSharedFileByToken")

	if token == "" {
		return nil, domain.ErrNotFound
	}
	file, err := uc.fileRepo.GetByShareToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !file.IsPublic {
		return nil, domain.ErrNotFound
	}
	if shareExpired(file) {
		return nil, fmt.Errorf("%w: share link expired", domain.ErrForbidden)
	}
	file.YourPermission = domain.FilePermissionPublic
	return file, nil
}

func (uc *StorageUsecase) IncrementDownloadCount(ctx context.Context, fileID string) error {
	return uc.fileRepo.IncrementDownloads(ctx, fileID)
}

func (uc *StorageUsecase) RenameFile(ctx context.Context, requesterID int64, fileID, newName string) (*domain.File, error) {
	logger.Info(ctx, "usecase.storage.RenameFile", "file_id", fileID)

	trimmed := strings.TrimSpace(newName)
	if trimmed == "" || len(trimmed) > maxFilenameLen {
		return nil, fmt.Errorf("%w: invalid filename length", domain.ErrInvalidInput)
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return nil, fmt.Errorf("%w: filename must not contain path separators", domain.ErrInvalidInput)
	}

	if err := uc.fileRepo.Rename(ctx, fileID, requesterID, trimmed); err != nil {
		return nil, err
	}

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	file.YourPermission = domain.FilePermissionOwner
	return file, nil
}

func (uc *StorageUsecase) GetDownloadable(ctx context.Context, fileID string, userID int64) (*domain.File, bool, error) {
	logger.Info(ctx, "usecase.storage.GetDownloadable", "file_id", fileID, "user_id", userID)

	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, false, err
	}
	perm := uc.computePermission(ctx, file, userID)
	file.YourPermission = perm
	switch perm {
	case domain.FilePermissionOwner, domain.FilePermissionDownload, domain.FilePermissionPublic:
		return file, true, nil
	default:
		return file, false, nil
	}
}
