package domain

import "time"

type Comment struct {
	ID        int64
	UserID    int64
	BlockID   int64
	Text      string
	CreatedAt time.Time
}
