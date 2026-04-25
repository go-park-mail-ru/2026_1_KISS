package domain

import "time"

type IssueCategory string

const (
	CategoryBug      IssueCategory = "bug"
	CategoryIdea     IssueCategory = "idea"
	CategoryProblem  IssueCategory = "problem"
	CategoryFeedback IssueCategory = "feedback"
)

type IssueStatus string

const (
	IssueStatusOpen   IssueStatus = "open"
	IssueStatusInWork IssueStatus = "in_progress"
	IssueStatusClosed IssueStatus = "closed"
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
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int64 `form:"user_ids"`
}

type IssueMessage struct {
	ID        int64
	IssueID   int64
	UserID    int64
	IsAdmin   bool
	Content   string
	CreatedAt time.Time
}

type IssueStats struct {
	Total      int64
	Open       int64
	InProgress int64
	Closed     int64
	Bug        int64
	Idea       int64
	Problem    int64
	Feedback   int64
}
