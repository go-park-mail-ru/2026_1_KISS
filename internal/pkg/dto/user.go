package dto

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type UserResponse struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		AvatarURL:   u.AvatarURL,
		Status:      u.Status,
		Description: u.Description,
		IsAdmin:     u.IsAdmin,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
