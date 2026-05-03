package domain

import "time"

type Comment struct {
	ID        int64
	UserID    int64
	Username  string
	BlockID   int64
	Text      string
	CreatedAt time.Time
}
