package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/payment/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/payment/yookassa"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type YooKassaClient interface {
	CreatePayment(ctx context.Context, idempotenceKey string, body yookassa.CreatePaymentRequest) (*yookassa.Payment, error)
	GetPayment(ctx context.Context, paymentID string) (*yookassa.Payment, error)
}

type PaymentUsecase struct {
	payments  repository.PaymentRepository
	plans     repository.PlanRepository
	subs      repository.SubscriptionRepository
	yookassa  YooKassaClient
	auth      AuthClient
	defaultRU string
}

func New(
	payments repository.PaymentRepository,
	plans repository.PlanRepository,
	subs repository.SubscriptionRepository,
	yk YooKassaClient,
	auth AuthClient,
	defaultReturnURL string,
) *PaymentUsecase {
	return &PaymentUsecase{
		payments:  payments,
		plans:     plans,
		subs:      subs,
		yookassa:  yk,
		auth:      auth,
		defaultRU: defaultReturnURL,
	}
}

func (uc *PaymentUsecase) ListPlans(ctx context.Context) ([]paymentdomain.Plan, error) {
	return uc.plans.List(ctx)
}

func (uc *PaymentUsecase) GetMySubscription(ctx context.Context, userID int64) (*paymentdomain.Subscription, error) {
	sub, err := uc.subs.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

type CreateInput struct {
	UserID    int64
	UserEmail string
	PlanName  string
	ReturnURL string
}

type CreateOutput struct {
	PaymentID         string
	ConfirmationToken string
	AmountKopeks      int64
	PlanName          string
}

func (uc *PaymentUsecase) CreateSubscriptionPayment(ctx context.Context, in CreateInput) (*CreateOutput, error) {
	logger.Info(ctx, "usecase.payment.Create", "user_id", in.UserID, "plan", in.PlanName)

	if in.PlanName != domain.PlanPro && in.PlanName != domain.PlanMax {
		return nil, fmt.Errorf("%w: only pro/max are purchasable", domain.ErrInvalidInput)
	}

	plan, err := uc.plans.GetByName(ctx, in.PlanName)
	if err != nil {
		return nil, fmt.Errorf("get plan: %w", err)
	}

	idempotenceKey := uuid.New().String()
	returnURL := in.ReturnURL
	if returnURL == "" {
		returnURL = uc.defaultRU
	}

	ykPayment, err := uc.yookassa.CreatePayment(ctx, idempotenceKey, yookassa.CreatePaymentRequest{
		Amount:  yookassa.Amount{Value: yookassa.KopeksToString(plan.PriceKopeks), Currency: "RUB"},
		Capture: true,
		Confirmation: yookassa.Confirmation{
			Type:      "embedded",
			ReturnURL: returnURL,
		},
		Description: fmt.Sprintf("Подписка %s (тест) — user %d", plan.Name, in.UserID),
		Metadata: map[string]string{
			"user_id":   fmt.Sprintf("%d", in.UserID),
			"plan_name": plan.Name,
		},
	})
	if err != nil {
		return nil, err
	}

	confirmationToken := ""
	if ykPayment.Confirmation != nil {
		confirmationToken = ykPayment.Confirmation.ConfirmationToken
	}

	dbPayment := &paymentdomain.Payment{
		UserID:            in.UserID,
		PlanID:            plan.ID,
		YooKassaPaymentID: ykPayment.ID,
		Status:            paymentdomain.StatusPending,
		AmountKopeks:      plan.PriceKopeks,
		Currency:          "RUB",
		ConfirmationToken: confirmationToken,
		IdempotenceKey:    idempotenceKey,
		Description:       fmt.Sprintf("Подписка %s", plan.Name),
	}
	if err := uc.payments.Create(ctx, dbPayment); err != nil {
		return nil, err
	}

	return &CreateOutput{
		PaymentID:         dbPayment.ID,
		ConfirmationToken: confirmationToken,
		AmountKopeks:      plan.PriceKopeks,
		PlanName:          plan.Name,
	}, nil
}

func (uc *PaymentUsecase) GetStatus(ctx context.Context, paymentID string, userID int64) (*paymentdomain.Payment, error) {
	logger.Info(ctx, "usecase.payment.GetStatus", "payment_id", paymentID, "user_id", userID)

	dbPayment, err := uc.payments.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if dbPayment.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if dbPayment.Status != paymentdomain.StatusPending && dbPayment.Status != paymentdomain.StatusWaitingForCapture {
		return dbPayment, nil
	}

	if dbPayment.YooKassaPaymentID == "" {
		return dbPayment, nil
	}

	ykPayment, err := uc.yookassa.GetPayment(ctx, dbPayment.YooKassaPaymentID)
	if err != nil {
		return dbPayment, nil
	}

	if ykPayment.Status != dbPayment.Status {
		if err := uc.applyStatus(ctx, dbPayment, ykPayment.Status); err != nil {
			return nil, err
		}
		dbPayment.Status = ykPayment.Status
	}
	return dbPayment, nil
}

type WebhookInput struct {
	YooKassaPaymentID string
	Status            string
}

func (uc *PaymentUsecase) HandleWebhook(ctx context.Context, in WebhookInput) error {
	logger.Info(ctx, "usecase.payment.Webhook", "yookassa_id", in.YooKassaPaymentID, "status", in.Status)

	if in.YooKassaPaymentID == "" {
		return fmt.Errorf("%w: missing yookassa payment id", domain.ErrInvalidInput)
	}

	dbPayment, err := uc.payments.GetByYooKassaID(ctx, in.YooKassaPaymentID)
	if err != nil {
		return err
	}

	ykPayment, err := uc.yookassa.GetPayment(ctx, in.YooKassaPaymentID)
	if err != nil {
		return err
	}

	if ykPayment.Status == dbPayment.Status && dbPayment.Status == paymentdomain.StatusSucceeded {
		return nil
	}

	return uc.applyStatus(ctx, dbPayment, ykPayment.Status)
}

func (uc *PaymentUsecase) applyStatus(ctx context.Context, dbPayment *paymentdomain.Payment, newStatus string) error {
	switch newStatus {
	case paymentdomain.StatusSucceeded:
		if err := uc.payments.MarkPaid(ctx, dbPayment.ID); err != nil {
			return err
		}
		return uc.activateSubscription(ctx, dbPayment)
	case paymentdomain.StatusCanceled:
		return uc.payments.UpdateStatus(ctx, dbPayment.ID, paymentdomain.StatusCanceled, dbPayment.YooKassaPaymentID)
	default:
		return uc.payments.UpdateStatus(ctx, dbPayment.ID, newStatus, dbPayment.YooKassaPaymentID)
	}
}

func (uc *PaymentUsecase) activateSubscription(ctx context.Context, dbPayment *paymentdomain.Payment) error {
	plan, err := uc.findPlanByID(ctx, dbPayment.PlanID)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Duration(plan.DurationDays) * 24 * time.Hour)

	if err := uc.subs.UpsertActive(ctx, dbPayment.UserID, plan.ID, plan.ExecutionQuota, expiresAt); err != nil {
		return err
	}

	if err := uc.auth.SetUserPlan(ctx, dbPayment.UserID, plan.Name, expiresAt.Unix()); err != nil {
		logger.Error(ctx, "usecase.payment.activateSubscription", "auth_error", err)
		return err
	}
	return nil
}

func (uc *PaymentUsecase) findPlanByID(ctx context.Context, planID int64) (*paymentdomain.Plan, error) {
	plans, err := uc.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range plans {
		if plans[i].ID == planID {
			return &plans[i], nil
		}
	}
	return nil, domain.ErrNotFound
}
