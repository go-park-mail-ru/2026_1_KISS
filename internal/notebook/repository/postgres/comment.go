package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, comment *domain.Comment) (int64, error) {
	start := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx, `
		WITH inserted AS (
			INSERT INTO comments (user_id, block_id, text)
			VALUES ($1, $2, $3)
			RETURNING id, user_id, block_id, text, created_at
		)
		SELECT i.id, i.user_id, u.username, i.block_id, i.text, i.created_at
		FROM inserted i
		JOIN users u ON u.id = i.user_id`,
		comment.UserID, comment.BlockID, comment.Text,
	).Scan(&id, &comment.UserID, &comment.Username, &comment.BlockID, &comment.Text, &comment.CreatedAt)
	if err != nil {
		logger.Error(ctx, "repo.comment.Create", "error", err, "duration", time.Since(start))
		return 0, err
	}
	logger.Info(ctx, "repo.comment.Create", "comment_id", id, "duration", time.Since(start))
	return id, nil
}

func (r *CommentRepository) GetByBlockID(ctx context.Context, blockID int64) ([]domain.Comment, error) {
	start := time.Now()
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.user_id, u.username, c.block_id, c.text, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.block_id = $1
		ORDER BY c.created_at ASC`,
		blockID,
	)
	if err != nil {
		logger.Error(ctx, "repo.comment.GetByBlockID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	comments, err := scanComments(rows)
	if err != nil {
		logger.Error(ctx, "repo.comment.GetByBlockID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.comment.GetByBlockID", "block_id", blockID, "count", len(comments), "duration", time.Since(start))
	return comments, nil
}

func (r *CommentRepository) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.Comment, error) {
	start := time.Now()
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.user_id, u.username, c.block_id, c.text, c.created_at
		FROM comments c
		JOIN blocks b ON b.id = c.block_id
		JOIN users u ON u.id = c.user_id
		WHERE b.notebook_id = $1
		ORDER BY c.block_id ASC, c.created_at ASC`,
		notebookID,
	)
	if err != nil {
		logger.Error(ctx, "repo.comment.GetByNotebookID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	comments, err := scanComments(rows)
	if err != nil {
		logger.Error(ctx, "repo.comment.GetByNotebookID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.comment.GetByNotebookID", "notebook_id", notebookID, "count", len(comments), "duration", time.Since(start))
	return comments, nil
}

func (r *CommentRepository) GetByID(ctx context.Context, commentID int64) (*domain.Comment, error) {
	start := time.Now()
	var c domain.Comment
	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.user_id, u.username, c.block_id, c.text, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = $1`,
		commentID,
	).Scan(&c.ID, &c.UserID, &c.Username, &c.BlockID, &c.Text, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		logger.Error(ctx, "repo.comment.GetByID", "error", domain.ErrNotFound, "duration", time.Since(start))
		return nil, domain.ErrNotFound
	}
	if err != nil {
		logger.Error(ctx, "repo.comment.GetByID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.comment.GetByID", "comment_id", commentID, "duration", time.Since(start))
	return &c, nil
}

func (r *CommentRepository) Delete(ctx context.Context, commentID int64) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, commentID)
	if err != nil {
		logger.Error(ctx, "repo.comment.Delete", "error", err, "duration", time.Since(start))
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.comment.Delete", "error", err, "duration", time.Since(start))
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.comment.Delete", "error", domain.ErrNotFound, "duration", time.Since(start))
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.comment.Delete", "comment_id", commentID, "duration", time.Since(start))
	return nil
}

func scanComments(rows *sql.Rows) ([]domain.Comment, error) {
	var comments []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.UserID, &c.Username, &c.BlockID, &c.Text, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}
