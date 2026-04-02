package runner

import (
	"context"
	"errors"
)

var (
	ErrContainerNotFound = errors.New("runner container not found")
	ErrContainerNotReady = errors.New("runner container is not ready")
)

type Manager interface {
	GetContainerAddress(ctx context.Context, sessionID string) (string, error)
	StartSession(ctx context.Context, sessionID string, language string) (string, error)
	StopSession(ctx context.Context, sessionID string) error
	CleanupSessions(ctx context.Context)
	Close() error
}
