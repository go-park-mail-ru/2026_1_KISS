package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

type RunnerHandler struct {
	client pb.RunnerServiceClient
}

func NewRunnerHandler(client pb.RunnerServiceClient) *RunnerHandler {
	return &RunnerHandler{client: client}
}

func (h *RunnerHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("POST /api/v1/runner/{notebook_id}", authMw(http.HandlerFunc(h.ExecuteFromPosition)))
	mux.Handle("POST /api/v1/runner/{notebook_id}/block", authMw(http.HandlerFunc(h.ExecuteBlock)))
	mux.Handle("POST /api/v1/runner/{notebook_id}/stop", authMw(http.HandlerFunc(h.StopSession)))
}

type executionResultResponse struct {
	BlockID    int64    `json:"block_id"`
	Position   int      `json:"position"`
	Stdout     []string `json:"stdout,omitempty"`
	Stderr     []string `json:"stderr,omitempty"`
	Result     string   `json:"result,omitempty"`
	Error      string   `json:"error,omitempty"`
	ExecutedAt string   `json:"executed_at"`
	Duration   string   `json:"duration"`
}

func (h *RunnerHandler) ExecuteFromPosition(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notebookID, err := strconv.ParseInt(r.PathValue("notebook_id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	var blockPos int
	if blockPosStr := r.URL.Query().Get("block_position"); blockPosStr != "" {
		blockPos, err = strconv.Atoi(blockPosStr)
		if err != nil {
			httputil.Error(w, http.StatusBadRequest, "invalid block_position")
			return
		}
	}

	resp, err := h.client.ExecuteFromPosition(r.Context(), &pb.ExecuteFromPositionRequest{
		NotebookId:    notebookID,
		BlockPosition: int32(blockPos), //nolint:gosec // block position fits int32
		UserId:        user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	results := make([]executionResultResponse, len(resp.GetResults()))
	for i, res := range resp.GetResults() {
		results[i] = protoResultToResponse(res)
	}
	httputil.JSON(w, http.StatusOK, results)
}

func (h *RunnerHandler) ExecuteBlock(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notebookID, err := strconv.ParseInt(r.PathValue("notebook_id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	var blockPos int
	if blockPosStr := r.URL.Query().Get("block_position"); blockPosStr != "" {
		blockPos, err = strconv.Atoi(blockPosStr)
		if err != nil {
			httputil.Error(w, http.StatusBadRequest, "invalid block_position")
			return
		}
	}

	resp, err := h.client.ExecuteBlock(r.Context(), &pb.ExecuteBlockRequest{
		NotebookId:    notebookID,
		BlockPosition: int32(blockPos), //nolint:gosec // block position fits int32
		UserId:        user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoResultToResponse(resp.GetResult()))
}

func (h *RunnerHandler) StopSession(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notebookID, err := strconv.ParseInt(r.PathValue("notebook_id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	_, err = h.client.StopSession(r.Context(), &pb.StopSessionRequest{
		NotebookId: notebookID,
		UserId:     user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

func protoResultToResponse(r *pb.BlockExecutionResult) executionResultResponse {
	return executionResultResponse{
		BlockID:    r.GetBlockId(),
		Position:   int(r.GetPosition()),
		Stdout:     r.GetStdout(),
		Stderr:     r.GetStderr(),
		Result:     r.GetResult(),
		Error:      r.GetError(),
		ExecutedAt: time.Unix(r.GetExecutedAt(), 0).Format(time.RFC3339),
		Duration:   time.Duration(r.GetDurationNs()).String(),
	}
}
