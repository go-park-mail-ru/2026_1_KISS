package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
)

type PlanRepo struct {
	db *sql.DB
}

func NewPlanRepository(db *sql.DB) *PlanRepo {
	return &PlanRepo{db: db}
}

func (r *PlanRepo) GetByName(ctx context.Context, name string) (*paymentdomain.Plan, error) {
	p := &paymentdomain.Plan{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, price, execution_quota, duration_days, created_at FROM subscription_plans WHERE name = $1`,
		name,
	).Scan(&p.ID, &p.Name, &p.PriceKopeks, &p.ExecutionQuota, &p.DurationDays, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get plan by name: %w", err)
	}
	return p, nil
}

func (r *PlanRepo) List(ctx context.Context) ([]paymentdomain.Plan, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, price, execution_quota, duration_days, created_at FROM subscription_plans ORDER BY price ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	var plans []paymentdomain.Plan
	for rows.Next() {
		var p paymentdomain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.PriceKopeks, &p.ExecutionQuota, &p.DurationDays, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}
