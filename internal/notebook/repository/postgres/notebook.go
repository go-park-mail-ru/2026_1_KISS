package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type NotebookRepo struct {
	db *sql.DB
}

func NewNotebookRepository(db *sql.DB) *NotebookRepo {
	return &NotebookRepo{db: db}
}

func (r *NotebookRepo) Create(ctx context.Context, notebook *domain.Notebook) (int64, error) {
	start := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO notebooks (owner_id, title) VALUES ($1, $2) RETURNING id, created_at, updated_at`,
		notebook.OwnerID, notebook.Title,
	).Scan(&id, &notebook.CreatedAt, &notebook.UpdatedAt)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.Create", "error", err, "duration", time.Since(start), "owner_id", notebook.OwnerID)
		return 0, err
	}
	logger.Info(ctx, "repo.notebooks.Create", "duration", time.Since(start), "notebook_id", id, "owner_id", notebook.OwnerID)
	return id, nil
}

func (r *NotebookRepo) GetByID(ctx context.Context, id int64) (*domain.Notebook, error) {
	start := time.Now()
	nb := &domain.Notebook{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, owner_id, title, is_public, created_at, updated_at FROM notebooks WHERE id = $1`,
		id,
	).Scan(&nb.ID, &nb.OwnerID, &nb.Title, &nb.IsPublic, &nb.CreatedAt, &nb.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.notebooks.GetByID", "error", domain.ErrNotFound, "duration", time.Since(start), "notebook_id", id)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.notebooks.GetByID", "error", err, "duration", time.Since(start), "notebook_id", id)
		return nil, err
	}
	logger.Info(ctx, "repo.notebooks.GetByID", "duration", time.Since(start), "notebook_id", id)
	return nb, nil
}

func (r *NotebookRepo) GetByOwnerID(ctx context.Context, ownerID int64, limit, offset int, search string) ([]domain.Notebook, error) {
	start := time.Now()
	search = likeEscaper.Replace(search)
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, title, is_public, created_at, updated_at
		 FROM notebooks
		 WHERE owner_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`,
		ownerID, search, limit, offset,
	)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.GetByOwnerID", "error", err, "duration", time.Since(start), "owner_id", ownerID)
		return nil, err
	}
	defer rows.Close()

	notebooks := []domain.Notebook{}
	for rows.Next() {
		var nb domain.Notebook
		if err := rows.Scan(&nb.ID, &nb.OwnerID, &nb.Title, &nb.IsPublic, &nb.CreatedAt, &nb.UpdatedAt); err != nil {
			logger.Error(ctx, "repo.notebooks.GetByOwnerID", "error", err, "duration", time.Since(start), "owner_id", ownerID)
			return nil, err
		}
		notebooks = append(notebooks, nb)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.notebooks.GetByOwnerID", "error", err, "duration", time.Since(start), "owner_id", ownerID)
		return nil, err
	}
	logger.Info(ctx, "repo.notebooks.GetByOwnerID", "duration", time.Since(start), "owner_id", ownerID, "count", len(notebooks))
	return notebooks, nil
}

func (r *NotebookRepo) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	result, err := r.db.ExecContext(ctx, `DELETE FROM notebooks WHERE id = $1`, id)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.Delete", "error", err, "duration", time.Since(start), "notebook_id", id)
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.notebooks.Delete", "error", err, "duration", time.Since(start), "notebook_id", id)
		return err
	}
	if rows == 0 {
		logger.Error(ctx, "repo.notebooks.Delete", "error", domain.ErrNotFound, "duration", time.Since(start), "notebook_id", id)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.notebooks.Delete", "duration", time.Since(start), "notebook_id", id)
	return nil
}

func (r *NotebookRepo) Update(ctx context.Context, notebook *domain.Notebook) error {
	start := time.Now()
	err := r.db.QueryRowContext(ctx,
		`UPDATE notebooks SET title = $1, is_public = $2, updated_at = NOW() WHERE id = $3 AND owner_id = $4 RETURNING updated_at`,
		notebook.Title, notebook.IsPublic, notebook.ID, notebook.OwnerID,
	).Scan(&notebook.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.notebooks.Update", "error", domain.ErrNotFound, "duration", time.Since(start), "notebook_id", notebook.ID)
			return domain.ErrNotFound
		}
		logger.Error(ctx, "repo.notebooks.Update", "error", err, "duration", time.Since(start), "notebook_id", notebook.ID)
		return err
	}
	logger.Info(ctx, "repo.notebooks.Update", "duration", time.Since(start), "notebook_id", notebook.ID)
	return nil
}

func (r *NotebookRepo) CountByOwnerID(ctx context.Context, ownerID int64, search string) (int, error) {
	start := time.Now()
	search = likeEscaper.Replace(search)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notebooks
		 WHERE owner_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%')`,
		ownerID, search,
	).Scan(&count)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.CountByOwnerID", "error", err, "duration", time.Since(start), "owner_id", ownerID)
		return 0, err
	}
	logger.Info(ctx, "repo.notebooks.CountByOwnerID", "duration", time.Since(start), "owner_id", ownerID, "count", count)
	return count, nil
}

func (r *NotebookRepo) ListAll(ctx context.Context, limit, offset int, search string) ([]domain.Notebook, error) {
	start := time.Now()
	search = likeEscaper.Replace(search)
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, title, is_public, created_at, updated_at
		 FROM notebooks
		 WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
		 ORDER BY id DESC
		 LIMIT $2 OFFSET $3`,
		search, limit, offset,
	)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.ListAll", "error", err, "duration", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	notebooks := []domain.Notebook{}
	for rows.Next() {
		var nb domain.Notebook
		if err := rows.Scan(&nb.ID, &nb.OwnerID, &nb.Title, &nb.IsPublic, &nb.CreatedAt, &nb.UpdatedAt); err != nil {
			logger.Error(ctx, "repo.notebooks.ListAll.scan", "error", err, "duration", time.Since(start))
			return nil, err
		}
		notebooks = append(notebooks, nb)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.notebooks.ListAll.rows", "error", err, "duration", time.Since(start))
		return nil, err
	}
	logger.Info(ctx, "repo.notebooks.ListAll", "duration", time.Since(start), "count", len(notebooks))
	return notebooks, nil
}

func (r *NotebookRepo) CountAll(ctx context.Context, search string) (int, error) {
	start := time.Now()
	search = likeEscaper.Replace(search)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notebooks
		 WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')`,
		search,
	).Scan(&count)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.CountAll", "error", err, "duration", time.Since(start))
		return 0, err
	}
	logger.Info(ctx, "repo.notebooks.CountAll", "duration", time.Since(start), "count", count)
	return count, nil
}

func (r *NotebookRepo) GetSharedWithUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, error) {
	start := time.Now()
	rows, err := r.db.QueryContext(ctx,
		`SELECT n.id, n.owner_id, u.username, n.title, n.is_public, n.created_at, n.updated_at
		 FROM notebooks n
		 JOIN file_permissions fp ON fp.notebook_id = n.id
		 JOIN users u ON u.id = n.owner_id
		 WHERE fp.user_id = $1
		 ORDER BY n.created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.GetSharedWithUser", "error", err, "duration", time.Since(start), "user_id", userID)
		return nil, err
	}
	defer rows.Close()

	var notebooks []domain.Notebook
	for rows.Next() {
		var nb domain.Notebook
		if err := rows.Scan(&nb.ID, &nb.OwnerID, &nb.OwnerUsername, &nb.Title, &nb.IsPublic, &nb.CreatedAt, &nb.UpdatedAt); err != nil {
			logger.Error(ctx, "repo.notebooks.GetSharedWithUser", "error", err, "duration", time.Since(start), "user_id", userID)
			return nil, err
		}
		notebooks = append(notebooks, nb)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.notebooks.GetSharedWithUser", "error", err, "duration", time.Since(start), "user_id", userID)
		return nil, err
	}
	logger.Info(ctx, "repo.notebooks.GetSharedWithUser", "duration", time.Since(start), "user_id", userID, "count", len(notebooks))
	return notebooks, nil
}

func (r *NotebookRepo) CountSharedWithUser(ctx context.Context, userID int64) (int, error) {
	start := time.Now()
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM file_permissions WHERE user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		logger.Error(ctx, "repo.notebooks.CountSharedWithUser", "error", err, "duration", time.Since(start), "user_id", userID)
		return 0, err
	}
	logger.Info(ctx, "repo.notebooks.CountSharedWithUser", "duration", time.Since(start), "user_id", userID, "count", count)
	return count, nil
}
