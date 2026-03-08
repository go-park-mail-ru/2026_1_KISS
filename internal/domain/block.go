package domain

import "time"

type Block struct {
	ID             int64
	NotebookID     int64
	Type           string
	Language       string
	Content        string
	Position       int
	ExecutionCount *int
	Outputs        []BlockOutput
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BlockOutput struct {
	ID         int64
	BlockID    int64
	Position   int
	OutputType string
	Content    string
	CreatedAt  time.Time
}
