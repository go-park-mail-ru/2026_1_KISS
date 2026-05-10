package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/payment/yookassa"
)

type fakePayments struct {
	created    *paymentdomain.Payment
	byID       map[string]*paymentdomain.Payment
	byYK       map[string]*paymentdomain.Payment
	statusUpd  []string
	markedPaid []string
}

func newFakePayments() *fakePayments {
	return &fakePayments{byID: map[string]*paymentdomain.Payment{}, byYK: map[string]*paymentdomain.Payment{}}
}

func (f *fakePayments) Create(_ context.Context, p *paymentdomain.Payment) error {
	p.ID = "internal-uuid"
	f.created = p
	f.byID[p.ID] = p
	if p.YooKassaPaymentID != "" {
		f.byYK[p.YooKassaPaymentID] = p
	}
	return nil
}
func (f *fakePayments) GetByID(_ context.Context, id string) (*paymentdomain.Payment, error) {
	if p, ok := f.byID[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakePayments) GetByYooKassaID(_ context.Context, id string) (*paymentdomain.Payment, error) {
	if p, ok := f.byYK[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakePayments) UpdateStatus(_ context.Context, id, status, _ string) error {
	f.statusUpd = append(f.statusUpd, id+":"+status)
	if p, ok := f.byID[id]; ok {
		p.Status = status
	}
	return nil
}
func (f *fakePayments) MarkPaid(_ context.Context, id string) error {
	f.markedPaid = append(f.markedPaid, id)
	if p, ok := f.byID[id]; ok {
		p.Status = paymentdomain.StatusSucceeded
	}
	return nil
}

type fakePlans struct{ list []paymentdomain.Plan }

func (f *fakePlans) GetByName(_ context.Context, name string) (*paymentdomain.Plan, error) {
	for i := range f.list {
		if f.list[i].Name == name {
			return &f.list[i], nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakePlans) List(_ context.Context) ([]paymentdomain.Plan, error) { return f.list, nil }

type fakeSubs struct{ upserted bool }

func (f *fakeSubs) UpsertActive(_ context.Context, _, _ int64, _ int32, _ time.Time) error {
	f.upserted = true
	return nil
}
func (f *fakeSubs) GetActiveByUser(_ context.Context, _ int64) (*paymentdomain.Subscription, error) {
	return nil, domain.ErrNotFound
}

type fakeYK struct {
	created   *yookassa.CreatePaymentRequest
	createID  string
	createTok string
	createErr error
	getStatus string
	getErr    error
}

func (f *fakeYK) CreatePayment(_ context.Context, _ string, body yookassa.CreatePaymentRequest) (*yookassa.Payment, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = &body
	return &yookassa.Payment{
		ID:           f.createID,
		Status:       paymentdomain.StatusPending,
		Amount:       body.Amount,
		Confirmation: &yookassa.Confirmation{Type: "embedded", ConfirmationToken: f.createTok},
	}, nil
}
func (f *fakeYK) GetPayment(_ context.Context, id string) (*yookassa.Payment, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &yookassa.Payment{ID: id, Status: f.getStatus}, nil
}

type fakeAuth struct {
	called bool
	plan   string
	user   int64
}

func (f *fakeAuth) SetUserPlan(_ context.Context, userID int64, plan string, _ int64) error {
	f.called = true
	f.user = userID
	f.plan = plan
	return nil
}

func newUC() (*PaymentUsecase, *fakePayments, *fakePlans, *fakeSubs, *fakeYK, *fakeAuth) {
	pays := newFakePayments()
	plans := &fakePlans{list: []paymentdomain.Plan{
		{ID: 1, Name: "pro", PriceKopeks: 99900, ExecutionQuota: 100000, DurationDays: 30},
		{ID: 2, Name: "max", PriceKopeks: 199900, ExecutionQuota: 999999, DurationDays: 30},
	}}
	subs := &fakeSubs{}
	yk := &fakeYK{createID: "yk-001", createTok: "tok-xyz"}
	auth := &fakeAuth{}
	uc := New(pays, plans, subs, yk, auth, "https://example.com/return")
	return uc, pays, plans, subs, yk, auth
}

func TestCreate_Pro_Success(t *testing.T) {
	uc, pays, _, _, yk, _ := newUC()

	out, err := uc.CreateSubscriptionPayment(context.Background(), CreateInput{
		UserID: 42, UserEmail: "u@e", PlanName: "pro",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ConfirmationToken != "tok-xyz" {
		t.Fatalf("unexpected token: %s", out.ConfirmationToken)
	}
	if out.AmountKopeks != 99900 {
		t.Fatalf("unexpected amount: %d", out.AmountKopeks)
	}
	if pays.created == nil || pays.created.YooKassaPaymentID != "yk-001" {
		t.Fatalf("payment not stored or yk id missing: %+v", pays.created)
	}
	if yk.created.Amount.Value != "999.00" {
		t.Fatalf("unexpected amount sent to yookassa: %s", yk.created.Amount.Value)
	}
}

func TestCreate_RejectsFreePlan(t *testing.T) {
	uc, _, _, _, _, _ := newUC()
	_, err := uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "free"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_UnknownPlan(t *testing.T) {
	uc, _, _, _, _, _ := newUC()
	_, err := uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "platinum"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetStatus_PollsAndMarkPaid(t *testing.T) {
	uc, pays, _, subs, yk, auth := newUC()

	_, err := uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "pro"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	createdID := pays.created.ID

	yk.getStatus = paymentdomain.StatusSucceeded

	p, err := uc.GetStatus(context.Background(), createdID, 42)
	if err != nil {
		t.Fatalf("getstatus error: %v", err)
	}
	if p.Status != paymentdomain.StatusSucceeded {
		t.Errorf("expected succeeded, got %s", p.Status)
	}
	if !subs.upserted {
		t.Errorf("subscription not upserted")
	}
	if !auth.called || auth.plan != "pro" || auth.user != 42 {
		t.Errorf("auth not called correctly: %+v", auth)
	}
	if len(pays.markedPaid) != 1 {
		t.Errorf("expected 1 markPaid, got %d", len(pays.markedPaid))
	}
}

func TestGetStatus_OtherUserForbidden(t *testing.T) {
	uc, pays, _, _, _, _ := newUC()
	_, _ = uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "pro"})

	_, err := uc.GetStatus(context.Background(), pays.created.ID, 99)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestWebhook_Succeeded(t *testing.T) {
	uc, pays, _, subs, yk, auth := newUC()
	_, _ = uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "max"})
	yk.getStatus = paymentdomain.StatusSucceeded

	if err := uc.HandleWebhook(context.Background(), WebhookInput{YooKassaPaymentID: pays.created.YooKassaPaymentID, Status: "succeeded"}); err != nil {
		t.Fatalf("webhook error: %v", err)
	}
	if !subs.upserted {
		t.Errorf("subscription not upserted")
	}
	if auth.plan != "max" {
		t.Errorf("expected plan max, got %s", auth.plan)
	}
}

func TestWebhook_IdempotentOnSucceeded(t *testing.T) {
	uc, pays, _, subs, yk, _ := newUC()
	_, _ = uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "pro"})
	yk.getStatus = paymentdomain.StatusSucceeded
	_ = uc.HandleWebhook(context.Background(), WebhookInput{YooKassaPaymentID: pays.created.YooKassaPaymentID, Status: "succeeded"})

	subs.upserted = false
	if err := uc.HandleWebhook(context.Background(), WebhookInput{YooKassaPaymentID: pays.created.YooKassaPaymentID, Status: "succeeded"}); err != nil {
		t.Fatalf("second webhook error: %v", err)
	}
	if subs.upserted {
		t.Errorf("subscription upserted twice (not idempotent)")
	}
}

func TestWebhook_MissingYKID(t *testing.T) {
	uc, _, _, _, _, _ := newUC()
	err := uc.HandleWebhook(context.Background(), WebhookInput{})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListPlans(t *testing.T) {
	uc, _, _, _, _, _ := newUC()
	plans, err := uc.ListPlans(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}

func TestGetMySubscription_None(t *testing.T) {
	uc, _, _, _, _, _ := newUC()
	_, err := uc.GetMySubscription(context.Background(), 42)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestWebhook_Canceled(t *testing.T) {
	uc, pays, _, _, yk, _ := newUC()
	_, _ = uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "pro"})
	yk.getStatus = paymentdomain.StatusCanceled

	if err := uc.HandleWebhook(context.Background(), WebhookInput{YooKassaPaymentID: pays.created.YooKassaPaymentID, Status: "canceled"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pays.created.Status != paymentdomain.StatusCanceled {
		t.Errorf("expected canceled, got %s", pays.created.Status)
	}
}

func TestGetStatus_AlreadySucceededSkipsYK(t *testing.T) {
	uc, pays, _, _, _, _ := newUC()
	_, _ = uc.CreateSubscriptionPayment(context.Background(), CreateInput{UserID: 42, PlanName: "pro"})
	pays.created.Status = paymentdomain.StatusSucceeded

	p, err := uc.GetStatus(context.Background(), pays.created.ID, 42)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if p.Status != paymentdomain.StatusSucceeded {
		t.Errorf("expected succeeded, got %s", p.Status)
	}
}
