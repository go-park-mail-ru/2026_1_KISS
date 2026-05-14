//go:generate go run go.uber.org/mock/mockgen -destination=../../../internal/mocks/file_repository_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/storage/repository FileRepository
//go:generate go run go.uber.org/mock/mockgen -destination=../../../internal/mocks/file_share_repository_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/storage/repository FileShareRepository
package repository

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id string) (*domain.File, error)
	GetByShareToken(ctx context.Context, token string) (*domain.File, error)
	ListByOwner(ctx context.Context, ownerID int64, category string, limit, offset int) ([]domain.File, int, error)
	Delete(ctx context.Context, id string) error
	DeleteByURL(ctx context.Context, url string) (string, error)
	GetStats(ctx context.Context) (*domain.StorageStats, error)
	GetStatsByOwner(ctx context.Context, ownerID int64) (*domain.StorageStats, error)
	ListAll(ctx context.Context, category string, ownerID int64, limit, offset int) ([]domain.File, int, error)
	SetPublic(ctx context.Context, fileID string, ownerID int64, isPublic bool, token *string, expiresAt *time.Time) error
	IncrementDownloads(ctx context.Context, fileID string) error
	Rename(ctx context.Context, fileID string, ownerID int64, newName string) error
}

type FileShareRepository interface {
	Upsert(ctx context.Context, share *domain.FileShare) error
	Delete(ctx context.Context, fileID string, userID int64) error
	GetByFileID(ctx context.Context, fileID string) ([]domain.FileShare, error)
	GetPermission(ctx context.Context, fileID string, userID int64) (*domain.FileShare, error)
	ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]domain.File, int, error)
}
