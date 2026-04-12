package runner_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/container"
	session_rep "github.com/go-park-mail-ru/2026_1_KISS/internal/runner/session_repository"
	"github.com/google/uuid"
)

var (
	ErrSessionNotStarted = errors.New("session not started")
)

type RunnerService interface {
	StartSession(ctx context.Context, notebookID int64) error
	ExecuteFromPosition(ctx context.Context, notebookID int64, startPosition int) ([]*domain.BlockExecutionResult, error)
	ExecuteBlock(ctx context.Context, notebookID int64, blockPosition int) (*domain.BlockExecutionResult, error)
	StopSession(ctx context.Context, notebookID int64) error
	StartIdleReaper(ctx context.Context)
}

type runnerService struct {
	runnerManager container.Manager
	sessionRepo   session_rep.ExecutionSessionRepository
	notebookRepo  repository.NotebookRepository
	blockRepo     repository.BlockRepository
	idleTimeout   time.Duration
}

func NewRunnerService(
	runnerManager container.Manager, sessionRepo session_rep.ExecutionSessionRepository,
	notebookRepo repository.NotebookRepository, blockRepo repository.BlockRepository,
	idleTimeout time.Duration,
) RunnerService {
	return &runnerService{
		runnerManager: runnerManager,
		sessionRepo:   sessionRepo,
		notebookRepo:  notebookRepo,
		blockRepo:     blockRepo,
		idleTimeout:   idleTimeout,
	}
}

// Создает NotebookSession и Кондейтер для него
func (s *runnerService) StartSession(ctx context.Context, notebookID int64) error {
	_, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
		if err != nil {
			return err
		}
		notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
		if err != nil {
			return err
		}

		sessionID := uuid.New().String()
		baseURL, err := s.runnerManager.StartSession(ctx, sessionID, "python")
		if err != nil {
			return err
		}

		_, err = s.sessionRepo.CreateSession(notebook, baseURL, sessionID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *runnerService) ExecuteFromPosition(ctx context.Context, notebookID int64, startPosition int) ([]*domain.BlockExecutionResult, error) {
	nSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		return nil, ErrSessionNotStarted
	}

	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return nil, err
	}

	execResults, err := nSession.ExecuteFromPosition(ctx, notebook, startPosition)
	if err != nil {
		return nil, err
	}
	return execResults, nil
}

func (s *runnerService) ExecuteBlock(ctx context.Context, notebookID int64, blockPosition int) (*domain.BlockExecutionResult, error) {
	notebookSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		return nil, ErrSessionNotStarted
	}
	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	execResult, err := notebookSession.ExecuteBlock(ctx, notebook.Blocks[blockPosition])
	if err != nil {
		return nil, err
	}
	return execResult, nil
}

func (s *runnerService) StopSession(ctx context.Context, notebookID int64) error {
	nSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		return nil
	}
	s.sessionRepo.DeleteSession(notebookID)
	if err := s.runnerManager.StopSession(ctx, nSession.GetSessionID()); err != nil {
		return err
	}
	return nil
}

// StartIdleReaper запускает фоновую горутину, которая каждую минуту проверяет сессии
// и убивает контейнеры, к которым не было обращений дольше idleTimeout.
func (s *runnerService) StartIdleReaper(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.evictIdleSessions(ctx)
		}
	}
}

func (s *runnerService) evictIdleSessions(ctx context.Context) {
	sessions := s.sessionRepo.ListSessions()
	now := time.Now()
	for notebookID, session := range sessions {
		fmt.Println("SUB IS: ", now.Sub(session.LastActivity()))
		fmt.Println("TIMEOUT: ", s.idleTimeout)
		if now.Sub(session.LastActivity()) < s.idleTimeout {
			continue
		}
		s.sessionRepo.DeleteSession(notebookID)
		err := s.runnerManager.StopSession(ctx, session.GetSessionID())
		if err != nil {
			fmt.Printf("idle reaper: ERROR failed to stop session %s (notebook %d): %v\n", session.GetSessionID(), notebookID, err)
			continue
		}
		fmt.Printf("idle reaper: stopped idle session %s (notebook %d)\n", session.GetSessionID(), notebookID)
	}
}
