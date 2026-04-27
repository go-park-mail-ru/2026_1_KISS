package domain

import "time"

const (
	PlanFree   = "free"
	PlanFreeze = "freeze"
	PlanPro    = "pro"
	PlanMax    = "max"
	PlanAdmin  = "admin"
)

var ValidPlans = map[string]bool{
	PlanFree:   true,
	PlanFreeze: true,
	PlanPro:    true,
	PlanMax:    true,
	PlanAdmin:  true,
}

type User struct {
	ID               int64
	Username         string
	Email            string
	PasswordHash     string
	AvatarURL        string
	Status           string
	Description      string
	IsVerified       bool
	IsAdmin          bool
	Plan             string
	LastActiveAt     *time.Time
	TotalTimeSeconds int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}