package runner_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"go.uber.org/mock/gomock"
)

func setup(t *testing.T) (
	*gomock.Controller,
	*mocks.MockManager,
	*mocks.MockExecutionSessionRepository,
	*mocks.MockNotebookRepository,
	*mocks.MockBlockRepository,
	RunnerService,
) {
	ctrl := gomock.NewController(t)
	mgr := mocks.NewMockManager(ctrl)
	sessRepo := mocks.NewMockExecutionSessionRepository(ctrl)
	nbRepo := mocks.NewMockNotebookRepository(ctrl)
	blkRepo := mocks.NewMockBlockRepository(ctrl)
	svc := NewRunnerService(mgr, sessRepo, nbRepo, blkRepo, 10*time.Minute)
	return ctrl, mgr, sessRepo, nbRepo, blkRepo, svc
}

func TestStartSession_AlreadyExists(t *testing.T) {
	ctrl, _, sessRepo, _, _, svc := setup(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(ctrl)
	sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)

	err := svc.StartSession(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartSession_NewSession(t *testing.T) {
	ctrl, mgr, sessRepo, nbRepo, blkRepo, svc := setup(t)
	defer ctrl.Finish()

	sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	nb := &domain.Notebook{ID: 1, OwnerID: 10, Title: "test"}
	nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nb, nil)

	blocks := []domain.Block{
		{ID: 100, NotebookID: 1, Content: "print('hello')", Position: 0},
	}
	blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return(blocks, nil)

	mgr.EXPECT().StartSession(gomock.Any(), gomock.Any(), "python").Return("http://runner:8080", nil)

	mockSession := mocks.NewMockNotebookSession(ctrl)
	sessRepo.EXPECT().CreateSession(gomock.Any(), "http://runner:8080", gomock.Any()).Return(mockSession, nil)

	err := svc.StartSession(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartSession_NotebookNotFound(t *testing.T) {
	ctrl, _, sessRepo, nbRepo, _, svc := setup(t)
	defer ctrl.Finish()

	sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)
	nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nil, errors.New("not found"))

	err := svc.StartSession(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartSession_GetBlocksError(t *testing.T) {
	ctrl, _, sessRepo, nbRepo, blkRepo, svc := setup(t)
	defer ctrl.Finish()

	sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)
	nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{ID: 1}, nil)
	blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return(nil, errors.New("db error"))

	err := svc.StartSession(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartSession_ManagerStartError(t *testing.T) {
	ctrl, mgr, sessRepo, nbRepo, blkRepo, svc := setup(t)
	defer ctrl.Finish()

	sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)
	nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{ID: 1}, nil)
	blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return([]domain.Block{}, nil)
	mgr.EXPECT().StartSession(gomock.Any(), gomock.Any(), "python").Return("", errors.New("docker error"))

	err := svc.StartSession(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteFromPosition_SessionNotFound(t *testing.T) {
	ctrl, _, sessRepo, _, _, svc := setup(t)
	defer ctrl.Finish()

	sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	_, err := svc.ExecuteFromPosition(context.Background(), 1, 0)
	if !errors.Is(err, ErrSessionNotStarted) {
		t.Fatalf("expected ErrSessionNotStarted, got %v", err)
	}
}

func TestExecuteFromPosition_Success(t *testing.T) {
	ctrl, _, sessRepo, nbRepo, blkRepo, svc := setup(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(ctrl)
	sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)

	nb := &domain.Notebook{ID: 1, OwnerID: 10}
	nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nb, nil)

	blocks := []domain.Block{
		{ID: 100, NotebookID: 1, Content: "print('hello')", Position: 0},
	}
	blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return(blocks, nil)

	expected := []*domain.BlockExecutionResult{
		{BlockID: 100, Position: 0, Stdout: []string{"hello"}},
	}
	mockSession.EXPECT().ExecuteFromPosition(gomock.Any(), gomock.Any(), 0).Return(expected, nil)

	results, err := svc.ExecuteFromPosition(context.Background(), 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].BlockID != 100 {
		t.Errorf("expected block ID 100, got %d", results[0].BlockID)
	}
}

func TestExecuteBlock_SessionNotFound(t *testing.T) {
	ctrl, _, sessRepo, _, _, svc := setup(t)
	defer ctrl.Finish()

	sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	_, err := svc.ExecuteBlock(context.Background(), 1, 0)
	if !errors.Is(err, ErrSessionNotStarted) {
		t.Fatalf("expected ErrSessionNotStarted, got %v", err)
	}
}

func TestExecuteBlock_Success(t *testing.T) {
	ctrl, _, sessRepo, nbRepo, blkRepo, svc := setup(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(ctrl)
	sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)

	nb := &domain.Notebook{ID: 1, OwnerID: 10}
	nbRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nb, nil)

	blocks := []domain.Block{
		{ID: 100, NotebookID: 1, Content: "x = 1", Position: 0},
	}
	blkRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return(blocks, nil)

	expected := &domain.BlockExecutionResult{BlockID: 100, Position: 0}
	mockSession.EXPECT().ExecuteBlock(gomock.Any(), blocks[0]).Return(expected, nil)

	result, err := svc.ExecuteBlock(context.Background(), 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BlockID != 100 {
		t.Errorf("expected block ID 100, got %d", result.BlockID)
	}
}

func TestStopSession_NoActiveSession(t *testing.T) {
	ctrl, _, sessRepo, _, _, svc := setup(t)
	defer ctrl.Finish()

	sessRepo.EXPECT().GetSession(int64(1)).Return(nil, false)

	err := svc.StopSession(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopSession_Success(t *testing.T) {
	ctrl, mgr, sessRepo, _, _, svc := setup(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(ctrl)
	sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)
	mockSession.EXPECT().GetSessionID().Return("session-123")
	sessRepo.EXPECT().DeleteSession(int64(1))
	mgr.EXPECT().StopSession(gomock.Any(), "session-123").Return(nil)

	err := svc.StopSession(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopSession_ManagerError(t *testing.T) {
	ctrl, mgr, sessRepo, _, _, svc := setup(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockNotebookSession(ctrl)
	sessRepo.EXPECT().GetSession(int64(1)).Return(mockSession, true)
	mockSession.EXPECT().GetSessionID().Return("session-123")
	sessRepo.EXPECT().DeleteSession(int64(1))
	mgr.EXPECT().StopSession(gomock.Any(), "session-123").Return(errors.New("stop failed"))

	err := svc.StopSession(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}
