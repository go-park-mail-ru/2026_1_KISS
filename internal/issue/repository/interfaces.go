//go:generate mockgen -destination=../../mocks/issue_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/issue/repository IssueRepository
//go:generate mockgen -destination=../../mocks/issue_message_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/issue/repository IssueMessageRepository

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
	Delete(ctx context.Context, id int64, userID int64) error
	AdminUpdateStatus(ctx context.Context, id int64, status domain.IssueStatus) error
	GetStats(ctx context.Context) (*domain.IssueStats, error)
}

type IssueMessageRepository interface {
	Create(ctx context.Context, msg *domain.IssueMessage) (int64, error)
	GetByIssueID(ctx context.Context, issueID int64) ([]domain.IssueMessage, error)
}
