//go:generate go run go.uber.org/mock/mockgen -destination=../../mocks/session_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/runner/session_repository ExecutionSessionRepository
package session_repository

import (
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/notebook_session"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/utils"
)

// In-memory (можно перенести в redis, но потом)
type ExecutionSessionRepository interface {
	CreateSession(notebook *domain.Notebook, runnerBaseURL string, sessionID string) (notebook_session.NotebookSession, error)
	GetSession(notebookID int64) (notebook_session.NotebookSession, bool)
	DeleteSession(notebookID int64)
	ListSessions() map[int64]notebook_session.NotebookSession
}

type executionSessionRepository struct {
	sessions    map[int64]notebook_session.NotebookSession
	sessionsMu  sync.RWMutex
	execTimeout time.Duration
}

func NewExecutionSessionRepository(execTimeout time.Duration) ExecutionSessionRepository {
	return &executionSessionRepository{
		sessions:    make(map[int64]notebook_session.NotebookSession),
		execTimeout: execTimeout,
	}
}

func (e *executionSessionRepository) CreateSession(notebook *domain.Notebook, runnerBaseURL string, sessionID string) (notebook_session.NotebookSession, error) {
	blockStates := make(map[int64]*domain.BlockState)
	for _, block := range notebook.Blocks {
		blockStates[block.ID] = &domain.BlockState{
			BlockID:   block.ID,
			Position:  block.Position,
			Hash:      utils.ComputeHash(block.Content),
			Executed:  false,
			UpdatedAt: time.Now(),
		}
	}

	session := notebook_session.NewNotebookSession(notebook.ID, sessionID, runnerBaseURL, -1, blockStates, e.execTimeout)
	e.sessionsMu.Lock()
	e.sessions[notebook.ID] = session
	e.sessionsMu.Unlock()

	return session, nil
}

func (e *executionSessionRepository) GetSession(notebookID int64) (notebook_session.NotebookSession, bool) {
	e.sessionsMu.RLock()
	defer e.sessionsMu.RUnlock()
	session, ok := e.sessions[notebookID]
	return session, ok
}

func (e *executionSessionRepository) DeleteSession(notebookID int64) {
	e.sessionsMu.Lock()
	delete(e.sessions, notebookID)
	e.sessionsMu.Unlock()
}

func (e *executionSessionRepository) ListSessions() map[int64]notebook_session.NotebookSession {
	e.sessionsMu.RLock()
	defer e.sessionsMu.RUnlock()
	result := make(map[int64]notebook_session.NotebookSession, len(e.sessions))
	for id, s := range e.sessions {
		result[id] = s
	}
	return result
}
