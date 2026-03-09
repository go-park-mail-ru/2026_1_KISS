package models

import "time"

type Comment struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	CellID    int64     `json:"cell_id" db:"cell_id"`
	Text      string    `json:"text" db:"text"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	User *User `json:"user,omitempty"`
}

type CreateCommentRequest struct {
	Text string `json:"text" validate:"required"`
}
