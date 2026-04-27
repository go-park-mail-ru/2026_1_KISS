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

type DayCount struct {
	Date  time.Time
	Count int64
}

type MonthCount struct {
	Month time.Time
	Count int64
}
