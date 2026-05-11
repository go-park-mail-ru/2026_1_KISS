package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

const oauthAccountColumns = `id, user_id, provider, provider_id, created_at`

type OAuthAccountRepo struct {
	db *sql.DB
}

func NewOAuthAccountRepository(db *sql.DB) *OAuthAccountRepo {
	return &OAuthAccountRepo{db: db}
}

func (r *OAuthAccountRepo) Create(ctx context.Context, acc *domain.OAuthAccount) (int64, error) {
	start := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_id) VALUES ($1, $2, $3) RETURNING id, created_at`,
		acc.UserID, acc.Provider, acc.ProviderID,
	).Scan(&id, &acc.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			logger.Error(ctx, "repo.oauth_accounts.Create", "error", domain.ErrConflict, "duration", time.Since(start), "provider", acc.Provider)
			return 0, domain.ErrConflict
		}
		logger.Error(ctx, "repo.oauth_accounts.Create", "error", err, "duration", time.Since(start), "provider", acc.Provider)
		return 0, err
	}
	acc.ID = id
	logger.Info(ctx, "repo.oauth_accounts.Create", "duration", time.Since(start), "id", id, "provider", acc.Provider, "user_id", acc.UserID)
	return id, nil
}

func (r *OAuthAccountRepo) GetByProviderID(ctx context.Context, provider, providerID string) (*domain.OAuthAccount, error) {
	start := time.Now()
	a := &domain.OAuthAccount{}
	err := r.db.QueryRowContext(ctx,
		`SELECT `+oauthAccountColumns+` FROM oauth_accounts WHERE provider = $1 AND provider_id = $2`,
		provider, providerID,
	).Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderID, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.oauth_accounts.GetByProviderID", "error", domain.ErrNotFound, "duration", time.Since(start), "provider", provider)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.oauth_accounts.GetByProviderID", "error", err, "duration", time.Since(start), "provider", provider)
		return nil, err
	}
	logger.Info(ctx, "repo.oauth_accounts.GetByProviderID", "duration", time.Since(start), "provider", provider, "user_id", a.UserID)
	return a, nil
}

func (r *OAuthAccountRepo) ListByUserID(ctx context.Context, userID int64) ([]domain.OAuthAccount, error) {
	start := time.Now()
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+oauthAccountColumns+` FROM oauth_accounts WHERE user_id = $1 ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.oauth_accounts.ListByUserID", "error", err, "duration", time.Since(start), "user_id", userID)
		return nil, err
	}
	defer rows.Close()

	var list []domain.OAuthAccount
	for rows.Next() {
		var a domain.OAuthAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderID, &a.CreatedAt); err != nil {
			logger.Error(ctx, "repo.oauth_accounts.ListByUserID", "error", err, "duration", time.Since(start), "user_id", userID)
			return nil, err
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.oauth_accounts.ListByUserID", "error", err, "duration", time.Since(start), "user_id", userID)
		return nil, err
	}
	logger.Info(ctx, "repo.oauth_accounts.ListByUserID", "duration", time.Since(start), "user_id", userID, "count", len(list))
	return list, nil
}

func (r *OAuthAccountRepo) DeleteByID(ctx context.Context, id, userID int64) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM oauth_accounts WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.oauth_accounts.DeleteByID", "error", err, "duration", time.Since(start), "id", id)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.oauth_accounts.DeleteByID", "error", err, "duration", time.Since(start), "id", id)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.oauth_accounts.DeleteByID", "error", domain.ErrNotFound, "duration", time.Since(start), "id", id)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.oauth_accounts.DeleteByID", "duration", time.Since(start), "id", id)
	return nil
}
