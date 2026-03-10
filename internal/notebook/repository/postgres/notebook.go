package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type NotebookRepo struct {
	db *sql.DB
}

func NewNotebookRepository(db *sql.DB) *NotebookRepo {
	return &NotebookRepo{db: db}
}

func (r *NotebookRepo) Create(ctx context.Context, notebook *domain.Notebook) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO notebooks (owner_id, title) VALUES ($1, $2) RETURNING id, created_at, updated_at`,
		notebook.OwnerID, notebook.Title,
	).Scan(&id, &notebook.CreatedAt, &notebook.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *NotebookRepo) GetByID(ctx context.Context, id int64) (*domain.Notebook, error) {
	nb := &domain.Notebook{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, owner_id, title, is_public, created_at, updated_at FROM notebooks WHERE id = $1`,
		id,
	).Scan(&nb.ID, &nb.OwnerID, &nb.Title, &nb.IsPublic, &nb.CreatedAt, &nb.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return nb, nil
}

func (r *NotebookRepo) GetByOwnerID(ctx context.Context, ownerID int64, limit, offset int) ([]domain.Notebook, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, title, is_public, created_at, updated_at FROM notebooks WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		ownerID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notebooks := []domain.Notebook{}
	for rows.Next() {
		var nb domain.Notebook
		if err := rows.Scan(&nb.ID, &nb.OwnerID, &nb.Title, &nb.IsPublic, &nb.CreatedAt, &nb.UpdatedAt); err != nil {
			return nil, err
		}
		notebooks = append(notebooks, nb)
	}
	return notebooks, rows.Err()
}

func (r *NotebookRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM notebooks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *NotebookRepo) Update(ctx context.Context, notebook *domain.Notebook) error {
	err := r.db.QueryRowContext(ctx,
		`UPDATE notebooks SET title = $1, is_public = $2, updated_at = NOW() WHERE id = $3 AND owner_id = $4 RETURNING updated_at`,
		notebook.Title, notebook.IsPublic, notebook.ID, notebook.OwnerID,
	).Scan(&notebook.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *NotebookRepo) CountByOwnerID(ctx context.Context, ownerID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notebooks WHERE owner_id = $1`,
		ownerID,
	).Scan(&count)
	return count, err
}
