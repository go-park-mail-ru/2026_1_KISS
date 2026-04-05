package domain

import "time"

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	AvatarURL    string
	Status       string
	Description  string
	IsVerified   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
