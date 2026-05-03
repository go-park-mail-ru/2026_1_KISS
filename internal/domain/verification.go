package domain

import "time"

type VerificationToken struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiresAt time.Time
}
