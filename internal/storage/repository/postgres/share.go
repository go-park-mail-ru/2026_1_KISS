package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type FileShareRepo struct {
	db *sql.DB
}

func NewFileShareRepository(db *sql.DB) *FileShareRepo {
	return &FileShareRepo{db: db}
}

func (r *FileShareRepo) Upsert(ctx context.Context, share *domain.FileShare) error {
	logger.Info(ctx, "repo.file_share.Upsert", "file_id", share.FileID, "user_id", share.UserID)

	err := r.db.QueryRowContext(ctx,
		`INSERT INTO file_shares (file_id, user_id, permission_level)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (file_id, user_id) DO UPDATE
		 SET permission_level = EXCLUDED.permission_level
		 RETURNING created_at`,
		share.FileID, share.UserID, share.Level,
	).Scan(&share.CreatedAt)
	if err != nil {
		logger.Error(ctx, "repo.file_share.Upsert", "error", err)
		return fmt.Errorf("upsert file share: %w", err)
	}
	return nil
}

func (r *FileShareRepo) Delete(ctx context.Context, fileID string, userID int64) error {
	logger.Info(ctx, "repo.file_share.Delete", "file_id", fileID, "user_id", userID)

	res, err := r.db.ExecContext(ctx,
		`DELETE FROM file_shares WHERE file_id = $1 AND user_id = $2`,
		fileID, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.file_share.Delete", "error", err)
		return fmt.Errorf("delete file share: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *FileShareRepo) GetByFileID(ctx context.Context, fileID string) ([]domain.FileShare, error) {
	logger.Info(ctx, "repo.file_share.GetByFileID", "file_id", fileID)

	rows, err := r.db.QueryContext(ctx,
		`SELECT fs.file_id, fs.user_id, COALESCE(u.email, ''), fs.permission_level, fs.created_at
		 FROM file_shares fs
		 LEFT JOIN users u ON u.id = fs.user_id
		 WHERE fs.file_id = $1
		 ORDER BY fs.created_at ASC`,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("query file shares: %w", err)
	}
	defer rows.Close()

	var shares []domain.FileShare
	for rows.Next() {
		var s domain.FileShare
		if err := rows.Scan(&s.FileID, &s.UserID, &s.Email, &s.Level, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan file share: %w", err)
		}
		shares = append(shares, s)
	}
	return shares, rows.Err()
}

func (r *FileShareRepo) GetPermission(ctx context.Context, fileID string, userID int64) (*domain.FileShare, error) {
	logger.Info(ctx, "repo.file_share.GetPermission", "file_id", fileID, "user_id", userID)

	var s domain.FileShare
	err := r.db.QueryRowContext(ctx,
		`SELECT file_id, user_id, permission_level, created_at
		 FROM file_shares WHERE file_id = $1 AND user_id = $2`,
		fileID, userID,
	).Scan(&s.FileID, &s.UserID, &s.Level, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		logger.Error(ctx, "repo.file_share.GetPermission", "error", err)
		return nil, fmt.Errorf("get permission: %w", err)
	}
	return &s, nil
}

func (r *FileShareRepo) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]domain.File, int, error) {
	logger.Info(ctx, "repo.file_share.ListByUserID", "user_id", userID)

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM file_shares WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count file shares: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT f.id, f.owner_id, f.notebook_id, f.category, f.filename, f.storage_key, f.url,
		        f.mime_type, f.size, f.created_at, f.is_public, f.share_token, f.share_expires_at,
		        f.downloads_count, fs.permission_level
		 FROM file_shares fs
		 INNER JOIN files f ON f.id = fs.file_id
		 WHERE fs.user_id = $1
		 ORDER BY fs.created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query shared files: %w", err)
	}
	defer rows.Close()

	var files []domain.File
	for rows.Next() {
		var f domain.File
		var cat string
		var token sql.NullString
		var expires sql.NullTime
		var level string
		if err := rows.Scan(
			&f.ID, &f.OwnerID, &f.NotebookID, &cat, &f.Filename, &f.StorageKey, &f.URL,
			&f.MIMEType, &f.Size, &f.CreatedAt, &f.IsPublic, &token, &expires, &f.DownloadsCount, &level,
		); err != nil {
			return nil, 0, fmt.Errorf("scan shared file: %w", err)
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
		f.YourPermission = level
		files = append(files, f)
	}
	return files, total, rows.Err()
}
