package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type IssueRepo struct {
	db *sql.DB
}

func (r *IssueRepo) GetByID(ctx context.Context, id int64) (*domain.Issue, error) {
	start := time.Now()
	issue := &domain.Issue{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, category, status, content, created_at, updated_at, user_id FROM issue WHERE id = $1`,
		id,
	).Scan(&issue.ID, &issue.Category, &issue.Status, &issue.Content, &issue.CreatedAt, &issue.UpdatedAt, &issue.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.issue.GetByID", "error", domain.ErrNotFound, "duration", time.Since(start), "issue_id", id)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.issue.GetByID", "error", err, "duration", time.Since(start), "issue_id", id)
		return nil, err
	}
	logger.Info(ctx, "repo.issue.GetByID", "duration", time.Since(start), "issue_id", id)
	return issue, nil
}

func (r *IssueRepo) GetAll(ctx context.Context, limit, offset int, filter *domain.IssueFilter) ([]domain.Issue, error) {
	start := time.Now()

	query := `SELECT id, category, status, content, created_at, updated_at, user_id FROM issue WHERE 1=1`
	args := []interface{}{}
	argCounter := 1

	if filter != nil {
		if filter.ID != 0 {
			query += fmt.Sprintf(" AND id = $%d", argCounter)
			args = append(args, filter.ID)
			argCounter++
		}
		if filter.Category != "" {
			query += fmt.Sprintf(" AND category = $%d", argCounter)
			args = append(args, filter.Category)
			argCounter++
		}
		if filter.Status != "" {
			query += fmt.Sprintf(" AND status = $%d", argCounter)
			args = append(args, filter.Status)
			argCounter++
		}
		if filter.UserID != 0 {
			query += fmt.Sprintf(" AND user_id = $%d", argCounter)
			args = append(args, filter.UserID)
			argCounter++
		}
		if filter.Content != "" {
			search := likeEscaper.Replace(filter.Content)
			query += fmt.Sprintf(" AND content ILIKE '%%' || $%d || '%%'", argCounter)
			args = append(args, search)
			argCounter++
		}
		if !filter.CreatedAt.IsZero() {
			query += fmt.Sprintf(" AND created_at >= $%d", argCounter)
			args = append(args, filter.CreatedAt)
			argCounter++
		}
		if !filter.UpdatedAt.IsZero() {
			query += fmt.Sprintf(" AND updated_at >= $%d", argCounter)
			args = append(args, filter.UpdatedAt)
			argCounter++
		}
	}

	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", argCounter, argCounter+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		logger.Error(ctx, "repo.issue.GetAll", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	issues := []domain.Issue{}
	for rows.Next() {
		var issue domain.Issue
		var categoryStr, statusStr string
		if err := rows.Scan(&issue.ID, &categoryStr, &statusStr, &issue.Content, &issue.CreatedAt, &issue.UpdatedAt, &issue.UserID); err != nil {
			logger.Error(ctx, "repo.issue.GetAll.scan", "error", err, "duration", time.Since(start))
			return nil, err
		}
		issue.Category = domain.IssueCategory(categoryStr)
		issue.Status = domain.IssueStatus(statusStr)
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.issue.GetAll.rows", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.issue.GetAll", "duration", time.Since(start), "count", len(issues))
	return issues, nil
}

func (r *IssueRepo) Create(ctx context.Context, issue *domain.Issue) (int64, error) {
	start := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO issue (category, status, content, user_id) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		issue.Category, issue.Status, issue.Content, issue.UserID,
	).Scan(&id, &issue.CreatedAt, &issue.UpdatedAt)
	if err != nil {
		logger.Error(ctx, "repo.issue.Create", "error", err, "duration", time.Since(start), "user_id", issue.UserID)
		return 0, err
	}
	logger.Info(ctx, "repo.issue.Create", "duration", time.Since(start), "issue_id", id, "user_id", issue.UserID)
	return id, nil
}

func (r *IssueRepo) Update(ctx context.Context, issue *domain.Issue) error {
	start := time.Now()
	err := r.db.QueryRowContext(ctx,
		`UPDATE issue SET category = $1, status = $2, content = $3, updated_at = NOW() WHERE id = $4 AND user_id = $5 RETURNING updated_at`,
		issue.Category, issue.Status, issue.Content, issue.ID, issue.UserID,
	).Scan(&issue.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.issue.Update", "error", domain.ErrNotFound, "duration", time.Since(start), "issue_id", issue.ID)
			return domain.ErrNotFound
		}
		logger.Error(ctx, "repo.issue.Update", "error", err, "duration", time.Since(start), "issue_id", issue.ID)
		return err
	}
	logger.Info(ctx, "repo.issue.Update", "duration", time.Since(start), "issue_id", issue.ID)
	return nil
}

func (r *IssueRepo) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	result, err := r.db.ExecContext(ctx, `DELETE FROM issue WHERE id = $1`, id)
	if err != nil {
		logger.Error(ctx, "repo.issue.Delete", "error", err, "duration", time.Since(start), "issue_id", id)
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.issue.Delete", "error", err, "duration", time.Since(start), "issue_id", id)
		return err
	}
	if rows == 0 {
		logger.Error(ctx, "repo.issue.Delete", "error", domain.ErrNotFound, "duration", time.Since(start), "issue_id", id)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.issue.Delete", "duration", time.Since(start), "issue_id", id)
	return nil
}
