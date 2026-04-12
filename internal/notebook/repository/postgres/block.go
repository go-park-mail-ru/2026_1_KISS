package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type BlockRepo struct {
	db *sql.DB
}

func NewBlockRepository(db *sql.DB) *BlockRepo {
	return &BlockRepo{db: db}
}

func (r *BlockRepo) Create(ctx context.Context, block *domain.Block) (int64, error) {
	start := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO blocks (notebook_id, type, language, content, position) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		block.NotebookID, block.Type, block.Language, block.Content, block.Position,
	).Scan(&id, &block.CreatedAt, &block.UpdatedAt)
	if err != nil {
		logger.Error(ctx, "repo.blocks.Create", "error", err, "duration", time.Since(start), "notebook_id", block.NotebookID)
		return 0, err
	}
	logger.Info(ctx, "repo.blocks.Create", "duration", time.Since(start), "block_id", id, "notebook_id", block.NotebookID)
	return id, nil
}

func (r *BlockRepo) GetByID(ctx context.Context, blockID int64) (*domain.Block, error) {
	start := time.Now()
	b := &domain.Block{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE id = $1`,
		blockID,
	).Scan(&b.ID, &b.NotebookID, &b.Type, &b.Language, &b.Content, &b.Position, &b.ExecutionCount, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.blocks.GetByID", "error", domain.ErrNotFound, "duration", time.Since(start), "block_id", blockID)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.blocks.GetByID", "error", err, "duration", time.Since(start), "block_id", blockID)
		return nil, err
	}
	b.Outputs = []domain.BlockOutput{}
	logger.Info(ctx, "repo.blocks.GetByID", "duration", time.Since(start), "block_id", blockID)
	return b, nil
}

func (r *BlockRepo) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.Block, error) {
	start := time.Now()
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE notebook_id = $1 ORDER BY position ASC`,
		notebookID,
	)
	if err != nil {
		logger.Error(ctx, "repo.blocks.GetByNotebookID", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
		return nil, err
	}
	defer rows.Close()

	blocks := []domain.Block{}
	for rows.Next() {
		var b domain.Block
		if err := rows.Scan(&b.ID, &b.NotebookID, &b.Type, &b.Language, &b.Content, &b.Position, &b.ExecutionCount, &b.CreatedAt, &b.UpdatedAt); err != nil {
			logger.Error(ctx, "repo.blocks.GetByNotebookID", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
			return nil, err
		}
		b.Outputs = []domain.BlockOutput{}
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.blocks.GetByNotebookID", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
		return nil, err
	}
	logger.Info(ctx, "repo.blocks.GetByNotebookID", "duration", time.Since(start), "notebook_id", notebookID, "count", len(blocks))
	return blocks, nil
}

func (r *BlockRepo) Update(ctx context.Context, block *domain.Block) error {
	start := time.Now()
	err := r.db.QueryRowContext(ctx,
		`UPDATE blocks SET content = $1, type = $2, language = $3, updated_at = NOW() WHERE id = $4 RETURNING updated_at`,
		block.Content, block.Type, block.Language, block.ID,
	).Scan(&block.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.blocks.Update", "error", domain.ErrNotFound, "duration", time.Since(start), "block_id", block.ID)
			return domain.ErrNotFound
		}
		logger.Error(ctx, "repo.blocks.Update", "error", err, "duration", time.Since(start), "block_id", block.ID)
		return err
	}
	logger.Info(ctx, "repo.blocks.Update", "duration", time.Since(start), "block_id", block.ID)
	return nil
}

func (r *BlockRepo) Delete(ctx context.Context, blockID int64) error {
	start := time.Now()
	tx, err := r.db.Begin()
	if err != nil {
		logger.Error(ctx, "repo.blocks.Delete", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var position int
	var notebookID int64
	err = tx.QueryRowContext(ctx,
		`SELECT position, notebook_id FROM blocks WHERE id = $1`,
		blockID,
	).Scan(&position, &notebookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.blocks.Delete", "error", domain.ErrNotFound, "duration", time.Since(start), "block_id", blockID)
			return domain.ErrNotFound
		}
		logger.Error(ctx, "repo.blocks.Delete", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM blocks WHERE id = $1`, blockID)
	if err != nil {
		logger.Error(ctx, "repo.blocks.Delete", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE blocks SET position = position - 1 WHERE notebook_id = $1 AND position > $2`,
		notebookID, position,
	)
	if err != nil {
		logger.Error(ctx, "repo.blocks.Delete", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "repo.blocks.Delete", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}
	logger.Info(ctx, "repo.blocks.Delete", "duration", time.Since(start), "block_id", blockID)
	return nil
}
