package domain

import "time"

type Notebook struct {
	ID             int64
	OwnerID        int64
	OwnerUsername  string
	Title          string
	IsPublic       bool
	Blocks         []Block
	CreatedAt      time.Time
	UpdatedAt      time.Time
	YourPermission string
}
