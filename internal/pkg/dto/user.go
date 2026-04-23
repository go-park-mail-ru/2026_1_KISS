package dto

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type UserResponse struct {
	ID               int64      `json:"id"`
	Username         string     `json:"username"`
	Email            string     `json:"email"`
	AvatarURL        string     `json:"avatar_url"`
	Status           string     `json:"status"`
	Description      string     `json:"description"`
	IsAdmin          bool       `json:"is_admin"`
	Plan             string     `json:"plan"`
	LastActiveAt     *time.Time `json:"last_active_at,omitempty"`
	TotalTimeSeconds int64      `json:"total_time_seconds"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func NewUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:               u.ID,
		Username:         u.Username,
		Email:            u.Email,
		AvatarURL:        u.AvatarURL,
		Status:           u.Status,
		Description:      u.Description,
		IsAdmin:          u.IsAdmin,
		Plan:             u.Plan,
		LastActiveAt:     u.LastActiveAt,
		TotalTimeSeconds: u.TotalTimeSeconds,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}
