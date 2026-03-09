package domain

import "time"

type SubscriptionPlan struct {
	ID             int64
	Name           string
	Price          int
	ExecutionQuota int
	DurationDays   int
	CreatedAt      time.Time
}

type UserSubscription struct {
	ID                 int64
	UserID             int64
	PlanID             int64
	ExecutionRemaining int
	StartedAt          time.Time
	ExpiresAt          time.Time
	CreatedAt          time.Time
}
