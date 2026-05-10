package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type PaymentRepo struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Create(ctx context.Context, p *paymentdomain.Payment) error {
	logger.Info(ctx, "repo.payment.Create", "user_id", p.UserID, "plan_id", p.PlanID)

	var ykID sql.NullString
	if p.YooKassaPaymentID != "" {
		ykID = sql.NullString{String: p.YooKassaPaymentID, Valid: true}
	}
	var ct sql.NullString
	if p.ConfirmationToken != "" {
		ct = sql.NullString{String: p.ConfirmationToken, Valid: true}
	}

	err := r.db.QueryRowContext(ctx,
		`INSERT INTO payments (user_id, plan_id, yookassa_payment_id, status, amount_kopeks, currency, confirmation_token, idempotence_key, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		p.UserID, p.PlanID, ykID, p.Status, p.AmountKopeks, p.Currency, ct, p.IdempotenceKey, p.Description,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		logger.Error(ctx, "repo.payment.Create", "error", err)
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

func (r *PaymentRepo) GetByID(ctx context.Context, id string) (*paymentdomain.Payment, error) {
	logger.Info(ctx, "repo.payment.GetByID", "id", id)

	p := &paymentdomain.Payment{}
	var ykID, ct sql.NullString
	var paidAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, plan_id, yookassa_payment_id, status, amount_kopeks, currency, confirmation_token, idempotence_key, description, created_at, updated_at, paid_at
		 FROM payments WHERE id = $1`, id,
	).Scan(
		&p.ID, &p.UserID, &p.PlanID, &ykID, &p.Status, &p.AmountKopeks, &p.Currency,
		&ct, &p.IdempotenceKey, &p.Description, &p.CreatedAt, &p.UpdatedAt, &paidAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		logger.Error(ctx, "repo.payment.GetByID", "error", err)
		return nil, fmt.Errorf("get payment: %w", err)
	}
	if ykID.Valid {
		p.YooKassaPaymentID = ykID.String
	}
	if ct.Valid {
		p.ConfirmationToken = ct.String
	}
	if paidAt.Valid {
		p.PaidAt = &paidAt.Time
	}
	return p, nil
}

func (r *PaymentRepo) GetByYooKassaID(ctx context.Context, yookassaID string) (*paymentdomain.Payment, error) {
	logger.Info(ctx, "repo.payment.GetByYooKassaID", "yookassa_id", yookassaID)

	p := &paymentdomain.Payment{}
	var ykID, ct sql.NullString
	var paidAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, plan_id, yookassa_payment_id, status, amount_kopeks, currency, confirmation_token, idempotence_key, description, created_at, updated_at, paid_at
		 FROM payments WHERE yookassa_payment_id = $1`, yookassaID,
	).Scan(
		&p.ID, &p.UserID, &p.PlanID, &ykID, &p.Status, &p.AmountKopeks, &p.Currency,
		&ct, &p.IdempotenceKey, &p.Description, &p.CreatedAt, &p.UpdatedAt, &paidAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		logger.Error(ctx, "repo.payment.GetByYooKassaID", "error", err)
		return nil, fmt.Errorf("get payment by yookassa id: %w", err)
	}
	if ykID.Valid {
		p.YooKassaPaymentID = ykID.String
	}
	if ct.Valid {
		p.ConfirmationToken = ct.String
	}
	if paidAt.Valid {
		p.PaidAt = &paidAt.Time
	}
	return p, nil
}

func (r *PaymentRepo) UpdateStatus(ctx context.Context, id, status, yookassaID string) error {
	logger.Info(ctx, "repo.payment.UpdateStatus", "id", id, "status", status)

	var ykID sql.NullString
	if yookassaID != "" {
		ykID = sql.NullString{String: yookassaID, Valid: true}
	}

	res, err := r.db.ExecContext(ctx,
		`UPDATE payments SET status = $1, yookassa_payment_id = COALESCE($2, yookassa_payment_id), updated_at = NOW() WHERE id = $3`,
		status, ykID, id,
	)
	if err != nil {
		logger.Error(ctx, "repo.payment.UpdateStatus", "error", err)
		return fmt.Errorf("update payment status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PaymentRepo) MarkPaid(ctx context.Context, id string) error {
	logger.Info(ctx, "repo.payment.MarkPaid", "id", id)

	res, err := r.db.ExecContext(ctx,
		`UPDATE payments SET status = 'succeeded', paid_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		logger.Error(ctx, "repo.payment.MarkPaid", "error", err)
		return fmt.Errorf("mark payment paid: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
