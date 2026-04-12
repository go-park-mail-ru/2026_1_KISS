package http

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
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// UpdateProfileRequest is the request body for updating user profile.
type UpdateProfileRequest struct {
	Username    string `json:"username"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

// ChangePasswordRequest is the request body for changing password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeEmailRequest is the request body for changing email.
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}
