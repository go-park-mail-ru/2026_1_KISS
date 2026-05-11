package domain

import "time"

type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
