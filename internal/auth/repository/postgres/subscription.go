package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
)

type SubscriptionViewRepo struct {
	db *sql.DB
}

func NewSubscriptionViewRepository(db *sql.DB) *SubscriptionViewRepo {
	return &SubscriptionViewRepo{db: db}
}

func (r *SubscriptionViewRepo) GetActive(ctx context.Context, userID int64) (*repository.ActiveSubscription, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT us.id, sp.name, us.expires_at
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1
		 ORDER BY us.expires_at DESC
		 LIMIT 1`,
		userID,
	)
	var s repository.ActiveSubscription
	if err := row.Scan(&s.ID, &s.PlanName, &s.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}
