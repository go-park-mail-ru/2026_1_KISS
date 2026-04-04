package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type PermissionRepository struct {
	db *sql.DB
}

func NewPermissionRepository(db *sql.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) Upsert(ctx context.Context, perm *domain.FilePermission) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO file_permissions (notebook_id, user_id, permission_level)
		VALUES ($1, $2, $3)
		ON CONFLICT (notebook_id, user_id)
		DO UPDATE SET permission_level = EXCLUDED.permission_level`,
		perm.NotebookID, perm.UserID, perm.PermissionLevel,
	)
	return err
}

func (r *PermissionRepository) Delete(ctx context.Context, notebookID, userID int64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM file_permissions WHERE notebook_id = $1 AND user_id = $2`,
		notebookID, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PermissionRepository) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.FilePermission, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT notebook_id, user_id, permission_level FROM file_permissions WHERE notebook_id = $1`,
		notebookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.FilePermission
	for rows.Next() {
		var p domain.FilePermission
		if err := rows.Scan(&p.NotebookID, &p.UserID, &p.PermissionLevel); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *PermissionRepository) GetPermission(ctx context.Context, notebookID, userID int64) (*domain.FilePermission, error) {
	var p domain.FilePermission
	err := r.db.QueryRowContext(ctx,
		`SELECT notebook_id, user_id, permission_level FROM file_permissions WHERE notebook_id = $1 AND user_id = $2`,
		notebookID, userID,
	).Scan(&p.NotebookID, &p.UserID, &p.PermissionLevel)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
