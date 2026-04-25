package domain

import "time"

type IssueCategory string

const (
	CategoryBug      IssueCategory = "BUG"
	CategoryIdea     IssueCategory = "IDEA"
	CategoryProblem  IssueCategory = "PROBLEM"
	CategoryFeedback IssueCategory = "FEEDBACK"
)

type IssueStatus string

const (
	IssueStatusOpen   IssueStatus = "OPEN"
	IssueStatusClosed IssueStatus = "CLOSED"
	IssueStatusInWork IssueStatus = "IN WORK"
)

type Issue struct {
	ID        int64
	Category  IssueCategory
	Status    IssueStatus
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int64
}

type IssueFilter struct {
	ID        int64         `form:"id"`
	Category  IssueCategory `form:"category"`
	Status    IssueStatus   `form:"status"`
	Content   string        `form:"content"`
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int64 `form:"user_ids"`
}
