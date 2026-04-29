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
	"github.com/lib/pq"
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
	outputsMap, err := r.GetOutputsByBlockIDs(ctx, []int64{blockID})
	if err != nil {
		logger.Error(ctx, "repo.blocks.GetByID", "error", err, "duration", time.Since(start), "block_id", blockID)
		return nil, err
	}
	b.Outputs = outputsMap[blockID]
	if b.Outputs == nil {
		b.Outputs = []domain.BlockOutput{}
	}
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
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.blocks.GetByNotebookID", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
		return nil, err
	}
	if len(blocks) > 0 {
		blockIDs := make([]int64, len(blocks))
		for i := range blocks {
			blockIDs[i] = blocks[i].ID
		}
		outputsMap, err := r.GetOutputsByBlockIDs(ctx, blockIDs)
		if err != nil {
			logger.Error(ctx, "repo.blocks.GetByNotebookID", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
			return nil, err
		}
		for i := range blocks {
			blocks[i].Outputs = outputsMap[blocks[i].ID]
			if blocks[i].Outputs == nil {
				blocks[i].Outputs = []domain.BlockOutput{}
			}
		}
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

func (r *BlockRepo) SaveOutputs(ctx context.Context, blockID int64, outputs []domain.BlockOutput) error {
	start := time.Now()
	tx, err := r.db.Begin()
	if err != nil {
		logger.Error(ctx, "repo.blocks.SaveOutputs", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `DELETE FROM block_outputs WHERE block_id = $1`, blockID)
	if err != nil {
		logger.Error(ctx, "repo.blocks.SaveOutputs", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}

	if len(outputs) > 0 {
		valueStrings := make([]string, 0, len(outputs))
		valueArgs := make([]interface{}, 0, len(outputs)*4)
		for i, o := range outputs {
			base := i * 4
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4))
			valueArgs = append(valueArgs, blockID, o.Position, o.OutputType, o.Content)
		}
		query := "INSERT INTO block_outputs (block_id, position, output_type, content) VALUES " + strings.Join(valueStrings, ", ") //nolint:gosec // query is built from parameterized placeholders only
		_, err = tx.ExecContext(ctx, query, valueArgs...)
		if err != nil {
			logger.Error(ctx, "repo.blocks.SaveOutputs", "error", err, "duration", time.Since(start), "block_id", blockID)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "repo.blocks.SaveOutputs", "error", err, "duration", time.Since(start), "block_id", blockID)
		return err
	}
	logger.Info(ctx, "repo.blocks.SaveOutputs", "duration", time.Since(start), "block_id", blockID, "count", len(outputs))
	return nil
}

func (r *BlockRepo) GetOutputsByBlockIDs(ctx context.Context, blockIDs []int64) (map[int64][]domain.BlockOutput, error) {
	start := time.Now()
	result := make(map[int64][]domain.BlockOutput)
	if len(blockIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs WHERE block_id = ANY($1) ORDER BY block_id, position ASC`,
		pq.Array(blockIDs),
	)
	if err != nil {
		logger.Error(ctx, "repo.blocks.GetOutputsByBlockIDs", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o domain.BlockOutput
		if err := rows.Scan(&o.ID, &o.BlockID, &o.Position, &o.OutputType, &o.Content, &o.CreatedAt); err != nil {
			logger.Error(ctx, "repo.blocks.GetOutputsByBlockIDs", "error", err, "duration", time.Since(start))
			return nil, err
		}
		result[o.BlockID] = append(result[o.BlockID], o)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.blocks.GetOutputsByBlockIDs", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.blocks.GetOutputsByBlockIDs", "duration", time.Since(start), "block_count", len(blockIDs))
	return result, nil
}

func (r *BlockRepo) CreateBatch(ctx context.Context, blocks []domain.Block) ([]int64, error) {
	start := time.Now()
	tx, err := r.db.Begin()
	if err != nil {
		logger.Error(ctx, "repo.blocks.CreateBatch", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	ids := make([]int64, len(blocks))
	for i := range blocks {
		b := &blocks[i]
		var id int64
		err := tx.QueryRowContext(ctx,
			`INSERT INTO blocks (notebook_id, type, language, content, position) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			b.NotebookID, b.Type, b.Language, b.Content, b.Position,
		).Scan(&id)
		if err != nil {
			logger.Error(ctx, "repo.blocks.CreateBatch", "error", err, "duration", time.Since(start), "position", b.Position)
			return nil, err
		}
		ids[i] = id
		b.ID = id
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "repo.blocks.CreateBatch", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.blocks.CreateBatch", "duration", time.Since(start), "count", len(blocks))
	return ids, nil
}

func (r *BlockRepo) ReorderBlocks(ctx context.Context, notebookID int64, blockIDs []int64) error {
	start := time.Now()
	tx, err := r.db.Begin()
	if err != nil {
		logger.Error(ctx, "repo.blocks.ReorderBlocks", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`UPDATE blocks SET position = -(position + 1) WHERE notebook_id = $1`,
		notebookID,
	)
	if err != nil {
		logger.Error(ctx, "repo.blocks.ReorderBlocks", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
		return err
	}

	for i, blockID := range blockIDs {
		_, err = tx.ExecContext(ctx,
			`UPDATE blocks SET position = $1, updated_at = NOW() WHERE id = $2 AND notebook_id = $3`,
			i, blockID, notebookID,
		)
		if err != nil {
			logger.Error(ctx, "repo.blocks.ReorderBlocks", "error", err, "duration", time.Since(start), "notebook_id", notebookID, "block_id", blockID)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error(ctx, "repo.blocks.ReorderBlocks", "error", err, "duration", time.Since(start), "notebook_id", notebookID)
		return err
	}
	logger.Info(ctx, "repo.blocks.ReorderBlocks", "duration", time.Since(start), "notebook_id", notebookID, "count", len(blockIDs))
	return nil
}
