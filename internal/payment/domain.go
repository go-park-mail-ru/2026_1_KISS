package payment

import "time"

const (
	StatusPending           = "pending"
	StatusSucceeded         = "succeeded"
	StatusCanceled          = "canceled"
	StatusWaitingForCapture = "waiting_for_capture"
)

type Plan struct {
	ID             int64
	Name           string
	PriceKopeks    int64
	ExecutionQuota int32
	DurationDays   int32
	CreatedAt      time.Time
}

type Payment struct {
	ID                string
	UserID            int64
	PlanID            int64
	YooKassaPaymentID string
	Status            string
	AmountKopeks      int64
	Currency          string
	ConfirmationToken string
	IdempotenceKey    string
	Description       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	PaidAt            *time.Time
}

type Subscription struct {
	ID                 int64
	UserID             int64
	PlanID             int64
	PlanName           string
	ExecutionRemaining int32
	StartedAt          time.Time
	ExpiresAt          time.Time
	CreatedAt          time.Time
}
