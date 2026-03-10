package http

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type CreateNotebookRequest struct {
	Title string `json:"title"`
}

type UpdateNotebookRequest struct {
	Title    string `json:"title"`
	IsPublic bool   `json:"is_public"`
}

type CreateBlockRequest struct {
	Type     string `json:"type"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

type UpdateBlockRequest struct {
	Type     string `json:"type"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

type BlockResponse struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Language  string    `json:"language"`
	Content   string    `json:"content"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

type NotebookResponse struct {
	ID        int64           `json:"id"`
	OwnerID   int64           `json:"owner_id"`
	Title     string          `json:"title"`
	IsPublic  bool            `json:"is_public"`
	Blocks    []BlockResponse `json:"blocks,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func NewNotebookResponse(nb *domain.Notebook) NotebookResponse {
	resp := NotebookResponse{
		ID:        nb.ID,
		OwnerID:   nb.OwnerID,
		Title:     nb.Title,
		IsPublic:  nb.IsPublic,
		CreatedAt: nb.CreatedAt,
		UpdatedAt: nb.UpdatedAt,
	}
	if len(nb.Blocks) > 0 {
		resp.Blocks = make([]BlockResponse, len(nb.Blocks))
		for i, b := range nb.Blocks {
			resp.Blocks[i] = BlockResponse{
				ID:        b.ID,
				Type:      b.Type,
				Language:  b.Language,
				Content:   b.Content,
				Position:  b.Position,
				CreatedAt: b.CreatedAt,
			}
		}
	}
	return resp
}

type NotebookListResponse struct {
	Notebooks []NotebookResponse `json:"notebooks"`
	Total     int                `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}

func NewNotebookListResponse(notebooks []domain.Notebook, total, limit, offset int) NotebookListResponse {
	items := make([]NotebookResponse, len(notebooks))
	for i := range notebooks {
		items[i] = NewNotebookResponse(&notebooks[i])
	}
	return NotebookListResponse{
		Notebooks: items,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}
}
