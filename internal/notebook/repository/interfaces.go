//go:generate mockgen -destination=../../mocks/notebook_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository NotebookRepository,BlockRepository
package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type NotebookRepository interface {
	Create(ctx context.Context, notebook *domain.Notebook) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Notebook, error)
	GetByOwnerID(ctx context.Context, ownerID int64, limit, offset int, search string) ([]domain.Notebook, error)
	Update(ctx context.Context, notebook *domain.Notebook) error
	Delete(ctx context.Context, id int64) error
	CountByOwnerID(ctx context.Context, ownerID int64, search string) (int, error)
}

type BlockRepository interface {
	Create(ctx context.Context, block *domain.Block) (int64, error)
	GetByID(ctx context.Context, blockID int64) (*domain.Block, error)
	GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.Block, error)
	Update(ctx context.Context, block *domain.Block) error
	Delete(ctx context.Context, blockID int64) error
}
