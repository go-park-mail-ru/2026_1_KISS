//go:generate mockgen -destination=../../mocks/issue_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/issue/repository IssueRepository

package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type IssueRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Issue, error)
	GetAll(ctx context.Context, limit, offset int, filter *domain.IssueFilter) ([]domain.Issue, error)
	Create(ctx context.Context, issue *domain.Issue) (int64, error)
	Update(ctx context.Context, issue *domain.Issue) error
	Delete(ctx context.Context, id int64) error
}
