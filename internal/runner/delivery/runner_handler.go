package delivery

import (
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service"
)

type RunnerHandler interface {
	RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware)
}

func NewRunnerHandler(runnerServ runner_service.RunnerService) RunnerHandler {
	return &runnerHandler{
		runnerServ: runnerServ,
	}
}

type runnerHandler struct {
	runnerServ runner_service.RunnerService
}

func (c *runnerHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	// Исполняет все блоки начиная с block_position (или с 0 если block_position не указан)
	// query {block_position}
	mux.Handle("GET /api/v1/runner/{notebook_id}", authMw(http.HandlerFunc(c.ExecuteFromPosition)))
	// Аналогично ExecuteFromPosition, но исполняет только 1 блок block_position
	mux.Handle("GET /api/v1/runner/{notebook_id}/block", authMw(http.HandlerFunc(c.ExecuteBlock)))
	mux.Handle("GET /api/v1/runner/{notebook_id}/stop", authMw(http.HandlerFunc(c.StopExecSession)))
}

func (c *runnerHandler) ExecuteFromPosition(w http.ResponseWriter, r *http.Request) {
	notebookID, err := strconv.ParseInt(r.PathValue("notebook_id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query()
	blockPosition, err := strconv.ParseInt(query.Get("block_position"), 10, 64)
	if err != nil {
		blockPosition = int64(0)
	}
	ctx := r.Context()
	if err := c.runnerServ.StartSession(ctx, notebookID); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	execResult, err := c.runnerServ.ExecuteFromPosition(ctx, notebookID, int(blockPosition))
	if err != nil && len(execResult) == 0 {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, execResult)
}

func (c *runnerHandler) ExecuteBlock(w http.ResponseWriter, r *http.Request) {
	notebookID, err := strconv.ParseInt(r.PathValue("notebook_id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query()
	blockPosition, err := strconv.ParseInt(query.Get("block_position"), 10, 64)
	if err != nil {
		blockPosition = int64(0)
	}
	ctx := r.Context()
	if err := c.runnerServ.StartSession(ctx, notebookID); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	execResult, err := c.runnerServ.ExecuteBlock(ctx, notebookID, int(blockPosition))
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, execResult)
}

func (c *runnerHandler) StopExecSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	notebookID, err := strconv.ParseInt(r.PathValue("notebook_id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.runnerServ.StopSession(ctx, notebookID); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, struct{}{})
}
