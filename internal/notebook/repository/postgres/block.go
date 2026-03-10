package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type BlockRepo struct {
	db *sql.DB
}

func NewBlockRepository(db *sql.DB) *BlockRepo {
	return &BlockRepo{db: db}
}

func (r *BlockRepo) Create(ctx context.Context, block *domain.Block) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO blocks (notebook_id, type, language, content, position) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		block.NotebookID, block.Type, block.Language, block.Content, block.Position,
	).Scan(&id, &block.CreatedAt, &block.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *BlockRepo) GetByID(ctx context.Context, blockID int64) (*domain.Block, error) {
	b := &domain.Block{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE id = $1`,
		blockID,
	).Scan(&b.ID, &b.NotebookID, &b.Type, &b.Language, &b.Content, &b.Position, &b.ExecutionCount, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	b.Outputs = []domain.BlockOutput{}
	return b, nil
}

func (r *BlockRepo) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.Block, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE notebook_id = $1 ORDER BY position ASC`,
		notebookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := []domain.Block{}
	for rows.Next() {
		var b domain.Block
		if err := rows.Scan(&b.ID, &b.NotebookID, &b.Type, &b.Language, &b.Content, &b.Position, &b.ExecutionCount, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		b.Outputs = []domain.BlockOutput{}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

func (r *BlockRepo) Update(ctx context.Context, block *domain.Block) error {
	err := r.db.QueryRowContext(ctx,
		`UPDATE blocks SET content = $1, type = $2, language = $3, updated_at = NOW() WHERE id = $4 RETURNING updated_at`,
		block.Content, block.Type, block.Language, block.ID,
	).Scan(&block.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *BlockRepo) Delete(ctx context.Context, blockID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var position int
	var notebookID int64
	err = tx.QueryRowContext(ctx,
		`SELECT position, notebook_id FROM blocks WHERE id = $1`,
		blockID,
	).Scan(&position, &notebookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM blocks WHERE id = $1`, blockID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE blocks SET position = position - 1 WHERE notebook_id = $1 AND position > $2`,
		notebookID, position,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
