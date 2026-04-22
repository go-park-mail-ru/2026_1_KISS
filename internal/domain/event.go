package domain

import (
	"encoding/json"
	"time"
)

type UserEvent struct {
	ID        int64
	UserID    int64
	EventType string
	Metadata  json.RawMessage
	CreatedAt time.Time
}
