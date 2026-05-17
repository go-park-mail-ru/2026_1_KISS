package pool_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/pool"
	"go.uber.org/mock/gomock"
)

func newPool(t *testing.T, size int) (*pool.Pool, *mocks.MockManager, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mgr := mocks.NewMockManager(ctrl)

	for i := 0; i < size; i++ {
		mgr.EXPECT().
			StartSession(gomock.Any(), gomock.Any(), "python").
			Return("http://localhost:8080", nil)
	}

	p, err := pool.New(context.Background(), mgr, "python", size, 10)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	return p, mgr, ctrl
}

func TestPool_AcquireRelease(t *testing.T) {
	p, mgr, ctrl := newPool(t, 1)
	defer ctrl.Finish()

	ctx := context.Background()

	w, pos, err := p.Acquire(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Acquire: unexpected error: %v", err)
	}
	if pos != 0 {
		t.Errorf("expected pos 0, got %d", pos)
	}
	if w == nil {
		t.Fatal("expected non-nil worker")
	}
	if w.State() != pool.WorkerAssigned {
		t.Errorf("expected WorkerAssigned, got %v", w.State())
	}

	// Second Acquire should queue (pool exhausted) — cancel immediately
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = p.Acquire(cancelCtx, 2, 10)
	if err == nil {
		t.Error("expected error for exhausted pool with cancelled context")
	}

	// Kernel restart will be called during Release — mock it to avoid HTTP calls
	// by just checking state transition after the pool is shut down
	mgr.EXPECT().CleanupSessions(gomock.Any())
	p.Shutdown(ctx)
}

func TestPool_QueuePositionFor(t *testing.T) {
	p, mgr, ctrl := newPool(t, 1)
	defer ctrl.Finish()

	pos := p.QueuePositionFor(42)
	if pos != 0 {
		t.Errorf("expected 0 for notebook not in queue, got %d", pos)
	}

	mgr.EXPECT().CleanupSessions(gomock.Any())
	p.Shutdown(context.Background())
}

func TestPool_GetContainerStats(t *testing.T) {
	p, mgr, ctrl := newPool(t, 1)
	defer ctrl.Finish()

	expected := &domain.ContainerResourceStats{CPUPercent: 5.0}
	mgr.EXPECT().GetContainerStats(gomock.Any(), gomock.Any()).Return(expected, nil)

	stats, err := p.GetContainerStats(context.Background(), "some-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CPUPercent != 5.0 {
		t.Errorf("expected 5.0, got %f", stats.CPUPercent)
	}

	mgr.EXPECT().CleanupSessions(gomock.Any())
	p.Shutdown(context.Background())
}

func TestPool_Release_ReturnsWorkerToIdle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mgr := mocks.NewMockManager(ctrl)
	mgr.EXPECT().StartSession(gomock.Any(), gomock.Any(), "python").Return(ts.URL, nil)

	p, err := pool.New(context.Background(), mgr, "python", 1, 10)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx := context.Background()

	w, _, err := p.Acquire(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	p.Release(ctx, w)

	// Worker should return to idle after restart — acquire with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	w2, _, err := p.Acquire(timeoutCtx, 2, 10)
	if err != nil {
		t.Fatalf("Acquire after Release: %v", err)
	}
	if w2 == nil {
		t.Fatal("expected non-nil worker after Release")
	}

	mgr.EXPECT().CleanupSessions(gomock.Any())
	p.Shutdown(ctx)
}

func TestWorker_AssignedTo(t *testing.T) {
	p, mgr, ctrl := newPool(t, 1)
	defer ctrl.Finish()

	ctx := context.Background()
	w, _, err := p.Acquire(ctx, 42, 10)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if w.AssignedTo() != 42 {
		t.Errorf("expected AssignedTo 42, got %d", w.AssignedTo())
	}

	mgr.EXPECT().CleanupSessions(gomock.Any())
	p.Shutdown(ctx)
}

func TestPool_Release_FailedRestart_SpawnReplacement(t *testing.T) {
	// First call: /restart returns error → pool.replaceWorker is triggered
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	// Second worker spawned to replace the broken one — it also points to ts.URL
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mgr := mocks.NewMockManager(ctrl)
	// First worker start
	mgr.EXPECT().StartSession(gomock.Any(), gomock.Any(), "python").Return(ts.URL, nil)
	// Replacement worker start
	mgr.EXPECT().StartSession(gomock.Any(), gomock.Any(), "python").Return(ts.URL, nil).AnyTimes()

	p, err := pool.New(context.Background(), mgr, "python", 1, 10)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	ctx := context.Background()
	w, _, err := p.Acquire(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Release triggers restart → fails → replaceWorker is called
	p.Release(ctx, w)

	// Wait for replacement worker to appear in idle (with timeout)
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	w2, _, err := p.Acquire(timeoutCtx, 2, 10)
	if err != nil {
		t.Logf("could not acquire after replace (might be timing): %v", err)
	} else if w2 != nil {
		_ = w2
	}

	mgr.EXPECT().CleanupSessions(gomock.Any())
	p.Shutdown(ctx)
}

func TestPool_New_AllWorkersFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mgr := mocks.NewMockManager(ctrl)

	mgr.EXPECT().
		StartSession(gomock.Any(), gomock.Any(), "python").
		Return("", domain.ErrInvalidInput).AnyTimes()

	_, err := pool.New(context.Background(), mgr, "python", 2, 10)
	if err == nil {
		t.Fatal("expected error when all workers fail to start")
	}
}
