//go:generate mockgen -destination=../../../internal/mocks/file_repository_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/storage/repository FileRepository
package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id string) (*domain.File, error)
	ListByOwner(ctx context.Context, ownerID int64, category string, limit, offset int) ([]domain.File, int, error)
	Delete(ctx context.Context, id string) error
	DeleteByURL(ctx context.Context, url string) (string, error)
	GetStats(ctx context.Context) (*domain.StorageStats, error)
	GetStatsByOwner(ctx context.Context, ownerID int64) (*domain.StorageStats, error)
	ListAll(ctx context.Context, category string, ownerID int64, limit, offset int) ([]domain.File, int, error)
}
