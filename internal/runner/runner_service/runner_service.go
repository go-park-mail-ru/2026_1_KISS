//go:generate go run go.uber.org/mock/mockgen -destination=../../mocks/runner_service_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service RunnerService
//go:generate go run go.uber.org/mock/mockgen -destination=../../mocks/worker_pool_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service WorkerPool
package runner_service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/pool"
	session_rep "github.com/go-park-mail-ru/2026_1_KISS/internal/runner/session_repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/snapshot"
)

var (
	ErrSessionNotStarted    = errors.New("session not started")
	ErrBlockPositionInvalid = errors.New("block position out of range")
)

// WorkerPool abstracts pool.Pool for testability.
type WorkerPool interface {
	Acquire(ctx context.Context, notebookID, userID int64) (*pool.Worker, int, error)
	Release(ctx context.Context, w *pool.Worker)
	GetContainerStats(ctx context.Context, workerID string) (*domain.ContainerResourceStats, error)
	QueuePositionFor(notebookID int64) int32
	Shutdown(ctx context.Context)
}

type RunnerService interface {
	StartSession(ctx context.Context, notebookID, userID int64) (queuePosition int, err error)
	ExecuteFromPosition(ctx context.Context, notebookID, userID int64, startPosition int) ([]*domain.BlockExecutionResult, error)
	ExecuteBlock(ctx context.Context, notebookID, userID int64, blockPosition int) (*domain.BlockExecutionResult, error)
	StopSession(ctx context.Context, notebookID, userID int64) error
	StartIdleReaper(ctx context.Context)
	GetSessionStats(ctx context.Context, notebookID, userID int64) (*domain.SessionStats, error)
	ExecuteBlockStreaming(ctx context.Context, notebookID, userID int64, blockPosition int, onChunk func(chunkType, data string)) (*domain.BlockExecutionResult, error)
}

type sessionWorker struct {
	worker *pool.Worker
	userID int64
}

type runnerService struct {
	workerPool   WorkerPool
	snapshotRepo snapshot.Repository
	sessionRepo  session_rep.ExecutionSessionRepository
	notebookRepo repository.NotebookRepository
	blockRepo    repository.BlockRepository
	idleTimeout  time.Duration
	httpClient   *http.Client

	workerMap   map[int64]*sessionWorker
	workerMapMu sync.RWMutex
}

func NewRunnerService(
	workerPool WorkerPool,
	snapshotRepo snapshot.Repository,
	sessionRepo session_rep.ExecutionSessionRepository,
	notebookRepo repository.NotebookRepository,
	blockRepo repository.BlockRepository,
	idleTimeout time.Duration,
) RunnerService {
	return &runnerService{
		workerPool:   workerPool,
		snapshotRepo: snapshotRepo,
		sessionRepo:  sessionRepo,
		notebookRepo: notebookRepo,
		blockRepo:    blockRepo,
		idleTimeout:  idleTimeout,
		httpClient:   &http.Client{Timeout: 35 * time.Second},
		workerMap:    make(map[int64]*sessionWorker),
	}
}

func (s *runnerService) StartSession(ctx context.Context, notebookID, userID int64) (int, error) {
	logger.Info(ctx, "usecase.runner.StartSession", "notebook_id", notebookID)

	if _, ok := s.sessionRepo.GetSession(notebookID); ok {
		return 0, nil
	}

	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return 0, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return 0, err
	}

	worker, pos, err := s.workerPool.Acquire(ctx, notebookID, userID)
	if err != nil {
		logger.Error(ctx, "usecase.runner.StartSession: acquire worker", "error", err)
		return pos, err
	}

	s.restoreIfExists(ctx, worker.BaseURL, notebookID, userID)

	if _, err := s.sessionRepo.CreateSession(notebook, worker.BaseURL, worker.ID); err != nil {
		s.workerPool.Release(ctx, worker)
		return 0, err
	}

	s.workerMapMu.Lock()
	s.workerMap[notebookID] = &sessionWorker{worker: worker, userID: userID}
	s.workerMapMu.Unlock()

	logger.Info(ctx, "usecase.runner.StartSession", "notebook_id", notebookID, "worker_id", worker.ID, "status", "ok")
	return pos, nil
}

func (s *runnerService) ExecuteFromPosition(ctx context.Context, notebookID, userID int64, startPosition int) ([]*domain.BlockExecutionResult, error) {
	logger.Info(ctx, "usecase.runner.ExecuteFromPosition", "notebook_id", notebookID, "start_position", startPosition)

	nSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		return nil, ErrSessionNotStarted
	}

	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, err
	}

	execResults, execErr := nSession.ExecuteFromPosition(ctx, notebook, startPosition)

	logger.Info(ctx, "usecase.runner.ExecuteFromPosition", "notebook_id", notebookID, "results_count", len(execResults))
	return execResults, execErr
}

func (s *runnerService) ExecuteBlock(ctx context.Context, notebookID, userID int64, blockPosition int) (*domain.BlockExecutionResult, error) {
	logger.Info(ctx, "usecase.runner.ExecuteBlock", "notebook_id", notebookID, "block_position", blockPosition)

	notebookSession, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		return nil, ErrSessionNotStarted
	}
	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, err
	}
	if blockPosition < 0 || blockPosition >= len(notebook.Blocks) {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, ErrBlockPositionInvalid
	}

	execResult, execErr := notebookSession.ExecuteBlock(ctx, notebook.Blocks[blockPosition])

	logger.Info(ctx, "usecase.runner.ExecuteBlock", "notebook_id", notebookID, "block_position", blockPosition, "status", "ok")
	return execResult, execErr
}

func (s *runnerService) ExecuteBlockStreaming(ctx context.Context, notebookID, userID int64, blockPosition int, onChunk func(chunkType, data string)) (*domain.BlockExecutionResult, error) {
	logger.Info(ctx, "usecase.runner.ExecuteBlockStreaming", "notebook_id", notebookID, "block_position", blockPosition)

	session, ok := s.sessionRepo.GetSession(notebookID)
	if !ok {
		return nil, ErrSessionNotStarted
	}
	notebook, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, err
	}
	notebook.Blocks, err = s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, err
	}
	if blockPosition < 0 || blockPosition >= len(notebook.Blocks) {
		s.releaseAfterExecution(context.Background(), notebookID)
		return nil, ErrBlockPositionInvalid
	}

	return session.ExecuteBlockStreaming(ctx, notebook.Blocks[blockPosition], onChunk)
}

func (s *runnerService) releaseAfterExecution(ctx context.Context, notebookID int64) {
	s.sessionRepo.DeleteSession(notebookID)

	s.workerMapMu.Lock()
	sw := s.workerMap[notebookID]
	delete(s.workerMap, notebookID)
	s.workerMapMu.Unlock()

	if sw != nil {
		s.workerPool.Release(ctx, sw.worker)
		logger.Info(ctx, "runner: worker released after execution", "notebook_id", notebookID, "worker_id", sw.worker.ID)
	}
}

func (s *runnerService) StopSession(ctx context.Context, notebookID, userID int64) error {
	logger.Info(ctx, "usecase.runner.StopSession", "notebook_id", notebookID)

	if _, ok := s.sessionRepo.GetSession(notebookID); !ok {
		return nil
	}
	s.sessionRepo.DeleteSession(notebookID)

	s.workerMapMu.Lock()
	sw := s.workerMap[notebookID]
	delete(s.workerMap, notebookID)
	s.workerMapMu.Unlock()

	if sw != nil {
		snapCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if snapErr := s.saveSnapshot(snapCtx, sw.worker.BaseURL, notebookID, sw.userID); snapErr != nil {
			logger.Error(ctx, "usecase.runner.StopSession: snapshot", "error", snapErr, "notebook_id", notebookID)
		}
		s.workerPool.Release(ctx, sw.worker)
	}
	logger.Info(ctx, "usecase.runner.StopSession", "notebook_id", notebookID, "status", "ok")
	return nil
}

func (s *runnerService) StartIdleReaper(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
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
		if now.Sub(session.LastActivity()) < s.idleTimeout {
			continue
		}
		s.sessionRepo.DeleteSession(notebookID)

		s.workerMapMu.Lock()
		sw := s.workerMap[notebookID]
		delete(s.workerMap, notebookID)
		s.workerMapMu.Unlock()

		if sw != nil {
			snapCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			if snapErr := s.saveSnapshot(snapCtx, sw.worker.BaseURL, notebookID, sw.userID); snapErr != nil {
				logger.Error(ctx, "idle_reaper: snapshot", "error", snapErr, "notebook_id", notebookID)
			}
			cancel()
			s.workerPool.Release(ctx, sw.worker)
			logger.Info(ctx, "idle_reaper: released worker", "notebook_id", notebookID, "worker_id", sw.worker.ID)
		}
	}
}

func (s *runnerService) GetSessionStats(ctx context.Context, notebookID, userID int64) (*domain.SessionStats, error) {
	stats := &domain.SessionStats{}

	s.workerMapMu.RLock()
	sw := s.workerMap[notebookID]
	s.workerMapMu.RUnlock()

	if sw != nil {
		if raw, err := s.workerPool.GetContainerStats(ctx, sw.worker.ID); err == nil {
			stats.ContainerResourceStats = *raw
		}
		stats.SessionState = "active"
	} else {
		stats.SessionState = "inactive"
		if queuePos := s.workerPool.QueuePositionFor(notebookID); queuePos > 0 {
			stats.SessionState = "queued"
			stats.QueuePosition = queuePos
		}
	}

	if exists, err := s.snapshotRepo.Exists(ctx, notebookID, userID); err == nil && exists {
		if _, meta, err := s.snapshotRepo.Load(ctx, notebookID, userID); err == nil {
			stats.SnapshotAge = time.Since(meta.SavedAt)
			stats.SnapshotSizeBytes = meta.SizeBytes
		}
	}

	return stats, nil
}

func (s *runnerService) saveSnapshot(ctx context.Context, baseURL string, notebookID, userID int64) error {
	logger.Info(ctx, "runner: saving snapshot", "notebook_id", notebookID)
	total := time.Now()

	dumpStart := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/snapshot", bytes.NewReader(nil))
	if err != nil {
		return fmt.Errorf("build snapshot request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /snapshot: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("snapshot returned %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Data == "" {
		logger.Info(ctx, "runner: snapshot empty, skipping save", "notebook_id", notebookID)
		return nil
	}

	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return fmt.Errorf("decode snapshot data: %w", err)
	}
	logger.Info(ctx, "runner: snapshot dumped", "notebook_id", notebookID, "size_bytes", len(data), "elapsed", time.Since(dumpStart))

	uploadStart := time.Now()
	if err := s.snapshotRepo.Save(ctx, notebookID, userID, data); err != nil {
		return err
	}
	logger.Info(ctx, "runner: snapshot saved", "notebook_id", notebookID, "size_bytes", len(data),
		"upload_elapsed", time.Since(uploadStart), "total_elapsed", time.Since(total))
	return nil
}

func (s *runnerService) restoreIfExists(ctx context.Context, baseURL string, notebookID, userID int64) {
	exists, err := s.snapshotRepo.Exists(ctx, notebookID, userID)
	if err != nil || !exists {
		return
	}
	downloadStart := time.Now()
	data, _, err := s.snapshotRepo.Load(ctx, notebookID, userID)
	if err != nil {
		logger.Error(ctx, "runner: load snapshot for restore", "error", err)
		return
	}
	logger.Info(ctx, "runner: snapshot downloaded", "notebook_id", notebookID, "size_bytes", len(data), "elapsed", time.Since(downloadStart))

	restoreStart := time.Now()
	encoded := base64.StdEncoding.EncodeToString(data)
	body, _ := json.Marshal(map[string]string{"data": encoded})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/restore", bytes.NewReader(body))
	if err != nil {
		logger.Error(ctx, "runner: build restore request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Error(ctx, "runner: POST /restore", "error", err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logger.Error(ctx, "runner: POST /restore non-200", "status", resp.StatusCode, "body", string(respBody), "notebook_id", notebookID)
		return
	}
	logger.Info(ctx, "runner: snapshot restored", "notebook_id", notebookID, "size_bytes", len(data), "elapsed", time.Since(restoreStart))
}
