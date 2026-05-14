package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

const fileSelectColumns = `id, owner_id, notebook_id, category, filename, storage_key, url, mime_type, size, created_at, is_public, share_token, share_expires_at, downloads_count`

type FileRepo struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) *FileRepo {
	return &FileRepo{db: db}
}

func (r *FileRepo) Create(ctx context.Context, file *domain.File) error {
	logger.Info(ctx, "repo.file.Create", "owner_id", file.OwnerID, "category", file.Category)

	err := r.db.QueryRowContext(ctx,
		`INSERT INTO files (owner_id, notebook_id, category, filename, storage_key, url, mime_type, size)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at`,
		file.OwnerID, file.NotebookID, string(file.Category),
		file.Filename, file.StorageKey, file.URL, file.MIMEType, file.Size,
	).Scan(&file.ID, &file.CreatedAt)
	if err != nil {
		logger.Error(ctx, "repo.file.Create", "error", err)
		return fmt.Errorf("insert file: %w", err)
	}
	return nil
}

func (r *FileRepo) GetByID(ctx context.Context, id string) (*domain.File, error) {
	logger.Info(ctx, "repo.file.GetByID", "id", id)

	row := r.db.QueryRowContext(ctx,
		`SELECT `+fileSelectColumns+` FROM files WHERE id = $1`, id,
	)
	f, err := scanFile(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		logger.Error(ctx, "repo.file.GetByID", "error", err)
		return nil, fmt.Errorf("get file: %w", err)
	}
	return f, nil
}

func (r *FileRepo) GetByShareToken(ctx context.Context, token string) (*domain.File, error) {
	logger.Info(ctx, "repo.file.GetByShareToken")

	row := r.db.QueryRowContext(ctx,
		`SELECT `+fileSelectColumns+` FROM files WHERE share_token = $1`, token,
	)
	f, err := scanFile(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		logger.Error(ctx, "repo.file.GetByShareToken", "error", err)
		return nil, fmt.Errorf("get file by share token: %w", err)
	}
	return f, nil
}

func (r *FileRepo) ListByOwner(ctx context.Context, ownerID int64, category string, limit, offset int) ([]domain.File, int, error) {
	logger.Info(ctx, "repo.file.ListByOwner", "owner_id", ownerID, "category", category)

	var total int
	if category != "" {
		err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM files WHERE owner_id = $1 AND category = $2`, ownerID, category,
		).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("count files: %w", err)
		}
	} else {
		err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM files WHERE owner_id = $1`, ownerID,
		).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("count files: %w", err)
		}
	}

	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT `+fileSelectColumns+` FROM files WHERE owner_id = $1 AND category = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			ownerID, category, limit, offset,
		)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT `+fileSelectColumns+` FROM files WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			ownerID, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	return scanFiles(rows, total)
}

func (r *FileRepo) Delete(ctx context.Context, id string) error {
	logger.Info(ctx, "repo.file.Delete", "id", id)

	res, err := r.db.ExecContext(ctx, `DELETE FROM files WHERE id = $1`, id)
	if err != nil {
		logger.Error(ctx, "repo.file.Delete", "error", err)
		return fmt.Errorf("delete file: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *FileRepo) DeleteByURL(ctx context.Context, url string) (string, error) {
	logger.Info(ctx, "repo.file.DeleteByURL", "url", url)

	var storageKey string
	err := r.db.QueryRowContext(ctx, `DELETE FROM files WHERE url = $1 RETURNING storage_key`, url).Scan(&storageKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		logger.Error(ctx, "repo.file.DeleteByURL", "error", err)
		return "", fmt.Errorf("delete file by url: %w", err)
	}
	return storageKey, nil
}

func (r *FileRepo) GetStats(ctx context.Context) (*domain.StorageStats, error) {
	logger.Info(ctx, "repo.file.GetStats")

	stats := &domain.StorageStats{
		FilesByCategory: make(map[domain.FileCategory]int64),
		SizeByCategory:  make(map[domain.FileCategory]int64),
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT category, COUNT(*), COALESCE(SUM(size), 0) FROM files GROUP BY category`,
	)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cat string
		var count, size int64
		if err := rows.Scan(&cat, &count, &size); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		fc := domain.FileCategory(cat)
		stats.FilesByCategory[fc] = count
		stats.SizeByCategory[fc] = size
		stats.TotalFiles += count
		stats.TotalSizeBytes += size
	}
	return stats, rows.Err()
}

func (r *FileRepo) GetStatsByOwner(ctx context.Context, ownerID int64) (*domain.StorageStats, error) {
	logger.Info(ctx, "repo.file.GetStatsByOwner", "owner_id", ownerID)

	stats := &domain.StorageStats{
		FilesByCategory: make(map[domain.FileCategory]int64),
		SizeByCategory:  make(map[domain.FileCategory]int64),
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT category, COUNT(*), COALESCE(SUM(size), 0) FROM files WHERE owner_id = $1 GROUP BY category`,
		ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get stats by owner: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cat string
		var count, size int64
		if err := rows.Scan(&cat, &count, &size); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		fc := domain.FileCategory(cat)
		stats.FilesByCategory[fc] = count
		stats.SizeByCategory[fc] = size
		stats.TotalFiles += count
		stats.TotalSizeBytes += size
	}
	return stats, rows.Err()
}

func (r *FileRepo) ListAll(ctx context.Context, category string, ownerID int64, limit, offset int) ([]domain.File, int, error) {
	logger.Info(ctx, "repo.file.ListAll", "category", category, "owner_id", ownerID)

	where := ` WHERE 1=1`
	filterArgs := []any{}
	idx := 1

	if category != "" {
		where += fmt.Sprintf(` AND category = $%d`, idx)
		filterArgs = append(filterArgs, category)
		idx++
	}
	if ownerID > 0 {
		where += fmt.Sprintf(` AND owner_id = $%d`, idx)
		filterArgs = append(filterArgs, ownerID)
		idx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM files`+where, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count files: %w", err)
	}

	selectQuery := fmt.Sprintf( //nolint:gosec
		`SELECT `+fileSelectColumns+` FROM files%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1,
	)
	selectArgs := append(filterArgs, limit, offset) //nolint:gocritic

	rows, err := r.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	return scanFiles(rows, total)
}

func (r *FileRepo) SetPublic(ctx context.Context, fileID string, ownerID int64, isPublic bool, token *string, expiresAt *time.Time) error {
	logger.Info(ctx, "repo.file.SetPublic", "id", fileID, "is_public", isPublic)

	res, err := r.db.ExecContext(ctx,
		`UPDATE files SET is_public = $1, share_token = $2, share_expires_at = $3
		 WHERE id = $4 AND owner_id = $5`,
		isPublic, token, expiresAt, fileID, ownerID,
	)
	if err != nil {
		logger.Error(ctx, "repo.file.SetPublic", "error", err)
		return fmt.Errorf("set public: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *FileRepo) IncrementDownloads(ctx context.Context, fileID string) error {
	logger.Info(ctx, "repo.file.IncrementDownloads", "id", fileID)

	res, err := r.db.ExecContext(ctx,
		`UPDATE files SET downloads_count = downloads_count + 1 WHERE id = $1`,
		fileID,
	)
	if err != nil {
		logger.Error(ctx, "repo.file.IncrementDownloads", "error", err)
		return fmt.Errorf("increment downloads: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *FileRepo) Rename(ctx context.Context, fileID string, ownerID int64, newName string) error {
	logger.Info(ctx, "repo.file.Rename", "id", fileID)

	res, err := r.db.ExecContext(ctx,
		`UPDATE files SET filename = $1 WHERE id = $2 AND owner_id = $3`,
		newName, fileID, ownerID,
	)
	if err != nil {
		logger.Error(ctx, "repo.file.Rename", "error", err)
		return fmt.Errorf("rename: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanFile(row *sql.Row) (*domain.File, error) {
	var f domain.File
	var cat string
	var token sql.NullString
	var expires sql.NullTime
	if err := row.Scan(
		&f.ID, &f.OwnerID, &f.NotebookID, &cat, &f.Filename, &f.StorageKey, &f.URL,
		&f.MIMEType, &f.Size, &f.CreatedAt, &f.IsPublic, &token, &expires, &f.DownloadsCount,
	); err != nil {
		return nil, err
	}
	f.Category = domain.FileCategory(cat)
	if token.Valid {
		v := token.String
		f.ShareToken = &v
	}
	if expires.Valid {
		v := expires.Time
		f.ShareExpiresAt = &v
	}
	return &f, nil
}

func scanFiles(rows *sql.Rows, total int) ([]domain.File, int, error) {
	var files []domain.File
	for rows.Next() {
		var f domain.File
		var cat string
		var token sql.NullString
		var expires sql.NullTime
		if err := rows.Scan(
			&f.ID, &f.OwnerID, &f.NotebookID, &cat, &f.Filename, &f.StorageKey, &f.URL,
			&f.MIMEType, &f.Size, &f.CreatedAt, &f.IsPublic, &token, &expires, &f.DownloadsCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan file: %w", err)
		}
		f.Category = domain.FileCategory(cat)
		if token.Valid {
			v := token.String
			f.ShareToken = &v
		}
		if expires.Valid {
			v := expires.Time
			f.ShareExpiresAt = &v
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	return files, total, nil
}
