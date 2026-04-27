package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/issue/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type issueMessageRepo struct {
	db *sql.DB
}

func NewIssueMessageRepository(db *sql.DB) repository.IssueMessageRepository {
	return &issueMessageRepo{db: db}
}

func (r *issueMessageRepo) Create(ctx context.Context, msg *domain.IssueMessage) (int64, error) {
	start := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO issue_message (issue_id, user_id, is_admin, content)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		msg.IssueID, msg.UserID, msg.IsAdmin, msg.Content,
	).Scan(&id, &msg.CreatedAt)
	if err != nil {
		logger.Error(ctx, "repo.issue_message.Create", "error", err, "duration", time.Since(start))
		return 0, err
	}
	logger.Info(ctx, "repo.issue_message.Create", "duration", time.Since(start), "id", id)
	return id, nil
}

func (r *issueMessageRepo) GetByIssueID(ctx context.Context, issueID int64) ([]domain.IssueMessage, error) {
	start := time.Now()
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, issue_id, user_id, is_admin, content, created_at
		 FROM issue_message
		 WHERE issue_id = $1
		 ORDER BY created_at ASC`,
		issueID,
	)
	if err != nil {
		logger.Error(ctx, "repo.issue_message.GetByIssueID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	msgs := []domain.IssueMessage{}
	for rows.Next() {
		var m domain.IssueMessage
		if err := rows.Scan(&m.ID, &m.IssueID, &m.UserID, &m.IsAdmin, &m.Content, &m.CreatedAt); err != nil {
			logger.Error(ctx, "repo.issue_message.GetByIssueID.scan", "error", err)
			return nil, err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.issue_message.GetByIssueID.rows", "error", err)
		return nil, err
	}
	logger.Info(ctx, "repo.issue_message.GetByIssueID", "duration", time.Since(start), "count", len(msgs))
	return msgs, nil
}
