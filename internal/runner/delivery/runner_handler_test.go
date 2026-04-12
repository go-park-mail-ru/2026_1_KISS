package delivery

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestExecuteFromPosition_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(nil)
	svc.EXPECT().ExecuteFromPosition(gomock.Any(), int64(1), 0).Return([]*domain.BlockExecutionResult{
		{BlockID: 100, Position: 0},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.ExecuteFromPosition(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestExecuteFromPosition_WithBlockPosition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(nil)
	svc.EXPECT().ExecuteFromPosition(gomock.Any(), int64(1), 2).Return([]*domain.BlockExecutionResult{
		{BlockID: 102, Position: 2},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1?block_position=2", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.ExecuteFromPosition(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestExecuteFromPosition_InvalidNotebookID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/abc", nil)
	req.SetPathValue("notebook_id", "abc")
	w := httptest.NewRecorder()

	h.ExecuteFromPosition(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExecuteFromPosition_StartSessionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(errors.New("start failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.ExecuteFromPosition(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestExecuteFromPosition_ExecuteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(nil)
	svc.EXPECT().ExecuteFromPosition(gomock.Any(), int64(1), 0).Return(nil, errors.New("exec failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.ExecuteFromPosition(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestExecuteBlock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(nil)
	svc.EXPECT().ExecuteBlock(gomock.Any(), int64(1), 0).Return(&domain.BlockExecutionResult{
		BlockID: 100, Position: 0,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1/block", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.ExecuteBlock(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestExecuteBlock_InvalidNotebookID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/abc/block", nil)
	req.SetPathValue("notebook_id", "abc")
	w := httptest.NewRecorder()

	h.ExecuteBlock(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExecuteBlock_StartSessionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(errors.New("start failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1/block", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.ExecuteBlock(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestExecuteBlock_ExecuteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(nil)
	svc.EXPECT().ExecuteBlock(gomock.Any(), int64(1), 0).Return(nil, errors.New("exec failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1/block", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.ExecuteBlock(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestStopExecSession_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StopSession(gomock.Any(), int64(1)).Return(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1/stop", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.StopExecSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStopExecSession_InvalidNotebookID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/abc/stop", nil)
	req.SetPathValue("notebook_id", "abc")
	w := httptest.NewRecorder()

	h.StopExecSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestStopExecSession_StopError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockRunnerService(ctrl)
	h := NewRunnerHandler(svc).(*runnerHandler)

	svc.EXPECT().StopSession(gomock.Any(), int64(1)).Return(errors.New("stop failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/1/stop", nil)
	req.SetPathValue("notebook_id", "1")
	w := httptest.NewRecorder()

	h.StopExecSession(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
