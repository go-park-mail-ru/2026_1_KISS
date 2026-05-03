package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type PermissionRepository struct {
	db *sql.DB
}

func NewPermissionRepository(db *sql.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) Upsert(ctx context.Context, perm *domain.FilePermission) error {
	start := time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO file_permissions (notebook_id, user_id, permission_level)
		VALUES ($1, $2, $3)
		ON CONFLICT (notebook_id, user_id)
		DO UPDATE SET permission_level = EXCLUDED.permission_level`,
		perm.NotebookID, perm.UserID, perm.PermissionLevel,
	)
	if err != nil {
		logger.Error(ctx, "repo.permissions.Upsert", "error", err, "duration", time.Since(start))
		return err
	}
	logger.Info(ctx, "repo.permissions.Upsert", "duration", time.Since(start), "notebook_id", perm.NotebookID, "user_id", perm.UserID)
	return nil
}

func (r *PermissionRepository) Delete(ctx context.Context, notebookID, userID int64) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM file_permissions WHERE notebook_id = $1 AND user_id = $2`,
		notebookID, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.permissions.Delete", "error", err, "duration", time.Since(start))
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.permissions.Delete", "error", err, "duration", time.Since(start))
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.permissions.Delete", "error", domain.ErrNotFound, "duration", time.Since(start))
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.permissions.Delete", "duration", time.Since(start), "notebook_id", notebookID, "user_id", userID)
	return nil
}

func (r *PermissionRepository) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.FilePermission, error) {
	start := time.Now()
	rows, err := r.db.QueryContext(ctx,
		`SELECT notebook_id, user_id, permission_level FROM file_permissions WHERE notebook_id = $1`,
		notebookID,
	)
	if err != nil {
		logger.Error(ctx, "repo.permissions.GetByNotebookID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	var perms []domain.FilePermission
	for rows.Next() {
		var p domain.FilePermission
		if err := rows.Scan(&p.NotebookID, &p.UserID, &p.PermissionLevel); err != nil {
			logger.Error(ctx, "repo.permissions.GetByNotebookID", "error", err, "duration", time.Since(start))
			return nil, err
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.permissions.GetByNotebookID", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.permissions.GetByNotebookID", "duration", time.Since(start), "notebook_id", notebookID, "count", len(perms))
	return perms, nil
}

func (r *PermissionRepository) GetPermission(ctx context.Context, notebookID, userID int64) (*domain.FilePermission, error) {
	start := time.Now()
	var p domain.FilePermission
	err := r.db.QueryRowContext(ctx,
		`SELECT notebook_id, user_id, permission_level FROM file_permissions WHERE notebook_id = $1 AND user_id = $2`,
		notebookID, userID,
	).Scan(&p.NotebookID, &p.UserID, &p.PermissionLevel)
	if errors.Is(err, sql.ErrNoRows) {
		logger.Error(ctx, "repo.permissions.GetPermission", "error", domain.ErrNotFound, "duration", time.Since(start))
		return nil, domain.ErrNotFound
	}
	if err != nil {
		logger.Error(ctx, "repo.permissions.GetPermission", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.permissions.GetPermission", "duration", time.Since(start), "notebook_id", notebookID, "user_id", userID)
	return &p, nil
}
