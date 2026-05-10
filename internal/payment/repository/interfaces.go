//go:generate go run go.uber.org/mock/mockgen -destination=../../../internal/mocks/payment_repository_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/payment/repository PaymentRepository,PlanRepository,SubscriptionRepository
package repository

import (
	"context"
	"time"

	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
)

type PaymentRepository interface {
	Create(ctx context.Context, p *paymentdomain.Payment) error
	GetByID(ctx context.Context, id string) (*paymentdomain.Payment, error)
	GetByYooKassaID(ctx context.Context, yookassaID string) (*paymentdomain.Payment, error)
	UpdateStatus(ctx context.Context, id, status, yookassaID string) error
	MarkPaid(ctx context.Context, id string) error
}

type PlanRepository interface {
	GetByName(ctx context.Context, name string) (*paymentdomain.Plan, error)
	List(ctx context.Context) ([]paymentdomain.Plan, error)
}

type SubscriptionRepository interface {
	UpsertActive(ctx context.Context, userID, planID int64, executionRemaining int32, expiresAt time.Time) error
	GetActiveByUser(ctx context.Context, userID int64) (*paymentdomain.Subscription, error)
}
