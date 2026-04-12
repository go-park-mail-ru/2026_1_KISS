//go:generate mockgen -destination=../../mocks/runner_service_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service RunnerService
package runner_service

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
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

func (s *runnerService) StartSession(ctx context.Context, notebookID int64) error {
	logger.Info(ctx, "usecase.runner.StartSession", "notebook_id", notebookID)

	_, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
		if err != nil {
			logger.Error(ctx, "usecase.runner.StartSession", "error", err)
			return err
		}
		notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
		if err != nil {
			logger.Error(ctx, "usecase.runner.StartSession", "error", err)
			return err
		}

		sessionID := uuid.New().String()
		baseURL, err := s.runnerManager.StartSession(ctx, sessionID, "python")
		if err != nil {
			logger.Error(ctx, "usecase.runner.StartSession", "error", err)
			return err
		}

		_, err = s.sessionRepo.CreateSession(notebook, baseURL, sessionID)
		if err != nil {
			logger.Error(ctx, "usecase.runner.StartSession", "error", err)
			return err
		}
	}
	logger.Info(ctx, "usecase.runner.StartSession", "notebook_id", notebookID, "status", "ok")
	return nil
}

func (s *runnerService) ExecuteFromPosition(ctx context.Context, notebookID int64, startPosition int) ([]*domain.BlockExecutionResult, error) {
	logger.Info(ctx, "usecase.runner.ExecuteFromPosition", "notebook_id", notebookID, "start_position", startPosition)

	nSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		logger.Error(ctx, "usecase.runner.ExecuteFromPosition", "error", ErrSessionNotStarted)
		return nil, ErrSessionNotStarted
	}

	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.runner.ExecuteFromPosition", "error", err)
		return nil, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.runner.ExecuteFromPosition", "error", err)
		return nil, err
	}

	execResults, err := nSession.ExecuteFromPosition(ctx, notebook, startPosition)
	if err != nil {
		logger.Error(ctx, "usecase.runner.ExecuteFromPosition", "error", err)
		return nil, err
	}
	logger.Info(ctx, "usecase.runner.ExecuteFromPosition", "notebook_id", notebookID, "results_count", len(execResults))
	return execResults, nil
}

func (s *runnerService) ExecuteBlock(ctx context.Context, notebookID int64, blockPosition int) (*domain.BlockExecutionResult, error) {
	logger.Info(ctx, "usecase.runner.ExecuteBlock", "notebook_id", notebookID, "block_position", blockPosition)

	notebookSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		logger.Error(ctx, "usecase.runner.ExecuteBlock", "error", ErrSessionNotStarted)
		return nil, ErrSessionNotStarted
	}
	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.runner.ExecuteBlock", "error", err)
		return nil, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.runner.ExecuteBlock", "error", err)
		return nil, err
	}
	execResult, err := notebookSession.ExecuteBlock(ctx, notebook.Blocks[blockPosition])
	if err != nil {
		logger.Error(ctx, "usecase.runner.ExecuteBlock", "error", err)
		return nil, err
	}
	logger.Info(ctx, "usecase.runner.ExecuteBlock", "notebook_id", notebookID, "block_position", blockPosition, "status", "ok")
	return execResult, nil
}

func (s *runnerService) StopSession(ctx context.Context, notebookID int64) error {
	logger.Info(ctx, "usecase.runner.StopSession", "notebook_id", notebookID)

	nSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		logger.Info(ctx, "usecase.runner.StopSession", "notebook_id", notebookID, "status", "no active session")
		return nil
	}
	s.sessionRepo.DeleteSession(notebookID)
	if err := s.runnerManager.StopSession(ctx, nSession.GetSessionID()); err != nil {
		logger.Error(ctx, "usecase.runner.StopSession", "error", err)
		return err
	}
	logger.Info(ctx, "usecase.runner.StopSession", "notebook_id", notebookID, "status", "ok")
	return nil
}
