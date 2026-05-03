package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

func TestRunnerHandler_RegisterRoutes(t *testing.T) {
	h := NewRunnerHandler(nil)
	mux := http.NewServeMux()
	identity := func(next http.Handler) http.Handler { return next }
	h.RegisterRoutes(mux, identity)
}

func TestRunnerHandler_StopSession_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockRunnerServiceClient(ctrl)

	client.EXPECT().StopSession(gomock.Any(), gomock.Any()).Return(&pb.StopSessionResponse{}, nil)

	h := NewRunnerHandler(client)
	req := httptest.NewRequest("POST", "/api/v1/runner/1/stop", nil)
	req.SetPathValue("notebook_id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.StopSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestRunnerHandler_ExecuteBlock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockRunnerServiceClient(ctrl)

	client.EXPECT().ExecuteBlock(gomock.Any(), gomock.Any()).Return(&pb.ExecuteBlockResponse{
		Result: &pb.BlockExecutionResult{
			BlockId:  10,
			Position: 0,
			Stdout:   []string{"hello"},
		},
	}, nil)

	h := NewRunnerHandler(client)
	req := httptest.NewRequest("POST", "/api/v1/runner/1/block?block_position=0", nil)
	req.SetPathValue("notebook_id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ExecuteBlock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestRunnerHandler_ExecuteFromPosition_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockRunnerServiceClient(ctrl)

	client.EXPECT().ExecuteFromPosition(gomock.Any(), gomock.Any()).Return(&pb.ExecuteFromPositionResponse{
		Results: []*pb.BlockExecutionResult{
			{BlockId: 10, Position: 0, Stdout: []string{"hello"}},
			{BlockId: 11, Position: 1},
		},
	}, nil)

	h := NewRunnerHandler(client)
	req := httptest.NewRequest("POST", "/api/v1/runner/1?block_position=0", nil)
	req.SetPathValue("notebook_id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ExecuteFromPosition(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestRunnerHandler_Unauthorized(t *testing.T) {
	h := NewRunnerHandler(nil)
	req := httptest.NewRequest("POST", "/api/v1/runner/1/stop", nil)
	req.SetPathValue("notebook_id", "1")
	rec := httptest.NewRecorder()

	h.StopSession(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestRunnerHandler_GetContainerStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockRunnerServiceClient(ctrl)

	client.EXPECT().GetSessionStats(gomock.Any(), gomock.Any()).
		Return(&pb.GetSessionStatsResponse{
			CpuPercent:    12.5,
			MemoryUsage:   134217728,
			MemoryLimit:   536870912,
			MemoryPercent: 25.0,
		}, nil)

	h := NewRunnerHandler(client)
	req := httptest.NewRequest("GET", "/api/v1/runner/1/stats", nil)
	req.SetPathValue("notebook_id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetContainerStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestRunnerHandler_GetContainerStats_Unauthorized(t *testing.T) {
	h := NewRunnerHandler(nil)
	req := httptest.NewRequest("GET", "/api/v1/runner/1/stats", nil)
	req.SetPathValue("notebook_id", "1")
	rec := httptest.NewRecorder()

	h.GetContainerStats(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestRunnerHandler_GetContainerStats_InvalidID(t *testing.T) {
	h := NewRunnerHandler(nil)
	req := httptest.NewRequest("GET", "/api/v1/runner/abc/stats", nil)
	req.SetPathValue("notebook_id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetContainerStats(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestRunnerHandler_ExecuteFromPosition_Unauthorized(t *testing.T) {
	h := NewRunnerHandler(nil)
	req := httptest.NewRequest("POST", "/api/v1/runner/1", nil)
	req.SetPathValue("notebook_id", "1")
	rec := httptest.NewRecorder()

	h.ExecuteFromPosition(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestRunnerHandler_ExecuteBlock_Unauthorized(t *testing.T) {
	h := NewRunnerHandler(nil)
	req := httptest.NewRequest("POST", "/api/v1/runner/1/block", nil)
	req.SetPathValue("notebook_id", "1")
	rec := httptest.NewRecorder()

	h.ExecuteBlock(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestRunnerHandler_ExecuteBlock_InvalidID(t *testing.T) {
	h := NewRunnerHandler(nil)
	req := httptest.NewRequest("POST", "/api/v1/runner/abc/block", nil)
	req.SetPathValue("notebook_id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ExecuteBlock(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}
