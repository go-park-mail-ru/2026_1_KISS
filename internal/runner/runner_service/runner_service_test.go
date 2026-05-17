package runner_service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	notebook_session "github.com/go-park-mail-ru/2026_1_KISS/internal/runner/notebook_session"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/pool"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/snapshot"
	"go.uber.org/mock/gomock"
)

type fixture struct {
	ctrl     *gomock.Controller
	wp       *mocks.MockWorkerPool
	snap     *mocks.MockRepository
	sessRepo *mocks.MockExecutionSessionRepository
	nbRepo   *mocks.MockNotebookRepository
	blkRepo  *mocks.MockBlockRepository
	svc      RunnerService
}

func setup(t *testing.T) fixture {
	ctrl := gomock.NewController(t)
	wp := mocks.NewMockWorkerPool(ctrl)
	snap := mocks.NewMockRepository(ctrl)
	sessRepo := mocks.NewMockExecutionSessionRepository(ctrl)
	nbRepo := mocks.NewMockNotebookRepository(ctrl)
	blkRepo := mocks.NewMockBlockRepository(ctrl)
	svc := NewRunnerService(wp, snap, sessRepo, nbRepo, blkRepo, 10*time.Minute)
	return fixture{ctrl, wp, snap, sessRepo, nbRepo, blkRepo, svc}
}

func TestStartSession_AlreadyExists(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(f.ctrl)
	f.sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)

	_, err := f.svc.StartSession(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartSession_NotebookNotFound(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nil, errors.New("not found"))

	_, err := f.svc.StartSession(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartSession_GetBlocksError(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{ID: 1}, nil)
	f.blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return(nil, errors.New("db error"))

	_, err := f.svc.StartSession(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartSession_PoolAcquireError(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{ID: 1}, nil)
	f.blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return([]domain.Block{}, nil)
	f.wp.EXPECT().Acquire(gomock.Any(), int64(1), int64(10)).Return(nil, 0, errors.New("pool full"))

	_, err := f.svc.StartSession(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartSession_NewSession(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)
	nb := &domain.Notebook{ID: 1, OwnerID: 10}
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nb, nil)
	blocks := []domain.Block{{ID: 100, NotebookID: 1, Content: "print('hello')", Position: 0}}
	f.blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return(blocks, nil)

	w := &pool.Worker{ID: "w-1", BaseURL: "http://runner:8080"}
	f.wp.EXPECT().Acquire(gomock.Any(), int64(1), int64(10)).Return(w, 0, nil)

	// restoreIfExists calls Exists — no snapshot
	f.snap.EXPECT().Exists(gomock.Any(), int64(1), int64(10)).Return(false, nil)

	mockSession := mocks.NewMockNotebookSession(f.ctrl)
	f.sessRepo.EXPECT().CreateSession(gomock.Any(), "http://runner:8080", "w-1").Return(mockSession, nil)

	pos, err := f.svc.StartSession(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pos != 0 {
		t.Errorf("expected queue position 0, got %d", pos)
	}
}

func TestExecuteFromPosition_SessionNotFound(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	_, err := f.svc.ExecuteFromPosition(context.Background(), 1, 10, 0)
	if !errors.Is(err, ErrSessionNotStarted) {
		t.Fatalf("expected ErrSessionNotStarted, got %v", err)
	}
}

func TestExecuteFromPosition_NotebookFetchError(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(f.ctrl)
	f.sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nil, errors.New("db error"))
	f.sessRepo.EXPECT().DeleteSession(int64(1))

	_, err := f.svc.ExecuteFromPosition(context.Background(), 1, 10, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteBlock_SessionNotFound(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	_, err := f.svc.ExecuteBlock(context.Background(), 1, 10, 0)
	if !errors.Is(err, ErrSessionNotStarted) {
		t.Fatalf("expected ErrSessionNotStarted, got %v", err)
	}
}

func TestExecuteBlock_InvalidPosition(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(f.ctrl)
	f.sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{ID: 1}, nil)
	f.blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return([]domain.Block{}, nil)
	f.sessRepo.EXPECT().DeleteSession(int64(1))

	_, err := f.svc.ExecuteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, ErrBlockPositionInvalid) {
		t.Fatalf("expected ErrBlockPositionInvalid, got %v", err)
	}
}

func TestExecuteBlock_Success(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(f.ctrl)
	f.sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)
	nb := &domain.Notebook{ID: 1, OwnerID: 10}
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nb, nil)
	blocks := []domain.Block{{ID: 100, NotebookID: 1, Content: "x = 1", Position: 0}}
	f.blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return(blocks, nil)

	expected := &domain.BlockExecutionResult{BlockID: 100, Position: 0}
	mockSession.EXPECT().ExecuteBlock(gomock.Any(), blocks[0]).Return(expected, nil)

	result, err := f.svc.ExecuteBlock(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BlockID != 100 {
		t.Errorf("expected block ID 100, got %d", result.BlockID)
	}
}

func TestStopSession_NoActiveSession(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	err := f.svc.StopSession(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopSession_Success(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(f.ctrl)
	f.sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)
	f.sessRepo.EXPECT().DeleteSession(int64(1))
	// No worker in workerMap for a fresh service, so Release is not called.

	err := f.svc.StopSession(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetSessionStats_NoWorker(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.wp.EXPECT().QueuePositionFor(int64(1)).Return(int32(0))
	f.snap.EXPECT().Exists(gomock.Any(), int64(1), int64(10)).Return(false, nil)

	stats, err := f.svc.GetSessionStats(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.SessionState != "inactive" {
		t.Errorf("expected inactive, got %s", stats.SessionState)
	}
}

func TestGetSessionStats_Queued(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.wp.EXPECT().QueuePositionFor(int64(5)).Return(int32(3))
	f.snap.EXPECT().Exists(gomock.Any(), int64(5), int64(10)).Return(false, nil)

	stats, err := f.svc.GetSessionStats(context.Background(), 5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.SessionState != "queued" {
		t.Errorf("expected queued, got %s", stats.SessionState)
	}
	if stats.QueuePosition != 3 {
		t.Errorf("expected queue position 3, got %d", stats.QueuePosition)
	}
}

func TestGetSessionStats_WithSnapshot(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.wp.EXPECT().QueuePositionFor(int64(1)).Return(int32(0))
	f.snap.EXPECT().Exists(gomock.Any(), int64(1), int64(10)).Return(true, nil)
	f.snap.EXPECT().Load(gomock.Any(), int64(1), int64(10)).Return([]byte("data"), snapshot.Metadata{SizeBytes: 42}, nil)

	stats, err := f.svc.GetSessionStats(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.SnapshotSizeBytes != 42 {
		t.Errorf("expected snapshot size 42, got %d", stats.SnapshotSizeBytes)
	}
}

func TestExecuteBlockStreaming_NoSession(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	_, err := f.svc.ExecuteBlockStreaming(context.Background(), 1, 10, 0, func(_, _ string) {})
	if !errors.Is(err, ErrSessionNotStarted) {
		t.Errorf("expected ErrSessionNotStarted, got %v", err)
	}
}

func TestEvictIdleSessions_NoWorker(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSess := mocks.NewMockNotebookSession(f.ctrl)
	mockSess.EXPECT().LastActivity().Return(time.Now().Add(-30 * time.Minute))

	sessions := map[int64]notebook_session.NotebookSession{
		int64(99): mockSess,
	}
	f.sessRepo.EXPECT().ListSessions().Return(sessions)
	f.sessRepo.EXPECT().DeleteSession(int64(99))

	svc := f.svc.(*runnerService)
	svc.evictIdleSessions(context.Background())
}

func TestEvictIdleSessions_NotIdle(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSess := mocks.NewMockNotebookSession(f.ctrl)
	mockSess.EXPECT().LastActivity().Return(time.Now())

	sessions := map[int64]notebook_session.NotebookSession{
		int64(99): mockSess,
	}
	f.sessRepo.EXPECT().ListSessions().Return(sessions)

	svc := f.svc.(*runnerService)
	svc.evictIdleSessions(context.Background())
}

func TestSaveSnapshot_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	f := setup(t)
	defer f.ctrl.Finish()

	svc := f.svc.(*runnerService)
	err := svc.saveSnapshot(context.Background(), ts.URL, 1, 10)
	if err == nil {
		t.Fatal("expected error from snapshot 500 response")
	}
}

func TestSaveSnapshot_Success(t *testing.T) {
	import_base64 := "aGVsbG8=" // base64("hello")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"` + import_base64 + `"}`))
	}))
	defer ts.Close()

	f := setup(t)
	defer f.ctrl.Finish()

	f.snap.EXPECT().Save(gomock.Any(), int64(1), int64(10), []byte("hello")).Return(nil)

	svc := f.svc.(*runnerService)
	err := svc.saveSnapshot(context.Background(), ts.URL, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreIfExists_NoSnapshot(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	f.snap.EXPECT().Exists(gomock.Any(), int64(1), int64(10)).Return(false, nil)

	svc := f.svc.(*runnerService)
	svc.restoreIfExists(context.Background(), "http://localhost:9999", 1, 10)
}

func TestRestoreIfExists_WithSnapshot(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	f := setup(t)
	defer f.ctrl.Finish()

	f.snap.EXPECT().Exists(gomock.Any(), int64(1), int64(10)).Return(true, nil)
	f.snap.EXPECT().Load(gomock.Any(), int64(1), int64(10)).Return([]byte("hello"), snapshot.Metadata{}, nil)

	svc := f.svc.(*runnerService)
	svc.restoreIfExists(context.Background(), ts.URL, 1, 10)
}

func TestExecuteBlockStreaming_InvalidPosition(t *testing.T) {
	f := setup(t)
	defer f.ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(f.ctrl)
	f.sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)
	f.nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{ID: 1}, nil)
	f.blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return([]domain.Block{}, nil)
	f.sessRepo.EXPECT().DeleteSession(int64(1))

	_, err := f.svc.ExecuteBlockStreaming(context.Background(), 1, 10, 5, func(_, _ string) {})
	if !errors.Is(err, ErrBlockPositionInvalid) {
		t.Errorf("expected ErrBlockPositionInvalid, got %v", err)
	}
}
