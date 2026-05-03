package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type VerificationRepo struct {
	db *sql.DB
}

func NewVerificationRepository(db *sql.DB) *VerificationRepo {
	return &VerificationRepo{db: db}
}

func (r *VerificationRepo) Create(ctx context.Context, vt *domain.VerificationToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO verification_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		vt.UserID, vt.Token, vt.ExpiresAt,
	)
	return err
}

func (r *VerificationRepo) GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error) {
	vt := &domain.VerificationToken{}

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token, expires_at FROM verification_tokens WHERE token=$1`,
		token,
	).Scan(&vt.ID, &vt.UserID, &vt.Token, &vt.ExpiresAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return vt, nil
}

func (r *VerificationRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM verification_tokens WHERE id=$1`,
		id,
	)
	return err
}
