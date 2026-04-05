package runner_service

import (
	"context"
	"errors"

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
}

type runnerService struct {
	runnerManager container.Manager
	sessionRepo   session_rep.ExecutionSessionRepository
	notebookRepo  repository.NotebookRepository
	blockRepo     repository.BlockRepository
}

func NewRunnerService(
	runnerManager container.Manager, sessionRepo session_rep.ExecutionSessionRepository,
	notebookRepo repository.NotebookRepository, blockRepo repository.BlockRepository,
) RunnerService {
	return &runnerService{
		runnerManager: runnerManager,
		sessionRepo:   sessionRepo,
		notebookRepo:  notebookRepo,
		blockRepo:     blockRepo,
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
		containerIP, err := s.runnerManager.StartSession(ctx, sessionID)
		if err != nil {
			return err
		}
		baseURL := "http://" + containerIP

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
