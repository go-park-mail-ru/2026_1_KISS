//go:generate go run go.uber.org/mock/mockgen -destination=../../mocks/snapshot_repository_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/runner/snapshot Repository
package snapshot

import (
	"context"
	"time"
)

type Metadata struct {
	FileID    string
	SavedAt   time.Time
	SizeBytes int64
	VarCount  int
}

type Repository interface {
	Save(ctx context.Context, notebookID, userID int64, data []byte) error
	Load(ctx context.Context, notebookID, userID int64) ([]byte, Metadata, error)
	Delete(ctx context.Context, notebookID, userID int64) error
	Exists(ctx context.Context, notebookID, userID int64) (bool, error)
}
