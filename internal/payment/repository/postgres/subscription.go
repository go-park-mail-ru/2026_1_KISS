package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
)

type SubscriptionRepo struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

func (r *SubscriptionRepo) UpsertActive(ctx context.Context, userID, planID int64, executionRemaining int32, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_subscriptions (user_id, plan_id, execution_remaining, started_at, expires_at)
		 VALUES ($1, $2, $3, NOW(), $4)`,
		userID, planID, executionRemaining, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert user_subscription: %w", err)
	}
	return nil
}

func (r *SubscriptionRepo) GetActiveByUser(ctx context.Context, userID int64) (*paymentdomain.Subscription, error) {
	s := &paymentdomain.Subscription{}
	err := r.db.QueryRowContext(ctx,
		`SELECT us.id, us.user_id, us.plan_id, sp.name, us.execution_remaining, us.started_at, us.expires_at, us.created_at
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1 AND us.expires_at > NOW()
		 ORDER BY us.expires_at DESC
		 LIMIT 1`,
		userID,
	).Scan(&s.ID, &s.UserID, &s.PlanID, &s.PlanName, &s.ExecutionRemaining, &s.StartedAt, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active subscription: %w", err)
	}
	return s, nil
}
