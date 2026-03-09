package models

import "time"

type SubscriptionPlan struct {
	ID             int64     `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Price          int       `json:"price" db:"price"`
	ExecutionQuota int       `json:"execution_quota" db:"execution_quota"`
	DurationDays   int       `json:"duration_days" db:"duration_day"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type UserSubscription struct {
	ID                 int64     `json:"id" db:"id"`
	UserID             int64     `json:"user_id" db:"user_id"`
	PlanID             int64     `json:"plan_id" db:"plan_id"`
	ExecutionRemaining int       `json:"execution_remaining" db:"execution_remaining"`
	StartedAt          time.Time `json:"started_at" db:"started_at"`
	ExpiresAt          time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`

	Plan *SubscriptionPlan `json:"plan,omitempty"`
}

type PurchaseSubscriptionRequest struct {
	PlanID int64 `json:"plan_id" validate:"required"`
}
