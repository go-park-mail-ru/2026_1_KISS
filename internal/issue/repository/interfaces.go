package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type IssueRepository interface {
	GetAll(ctx context.Context, limit, offset int, filter *domain.IssueFilter) ([]domain.Notebook, error)
	GetByID(ctx context.Context, id int64) (*domain.Notebook, error)
	Create(ctx context.Context, issue *domain.Issue) (int64, error)
	Update(ctx context.Context, issue *domain.Issue) error
	Delete(ctx context.Context, id int64) error
}
