package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pbstorage "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type StatsHandler struct {
	authClient     pbauth.AuthServiceClient
	notebookClient pbnotebook.NotebookServiceClient
	storageClient  pbstorage.StorageServiceClient
}

func NewStatsHandler(authClient pbauth.AuthServiceClient, notebookClient pbnotebook.NotebookServiceClient, storageClient pbstorage.StorageServiceClient) *StatsHandler {
	return &StatsHandler{authClient: authClient, notebookClient: notebookClient, storageClient: storageClient}
}

func (h *StatsHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("GET /api/v1/users/me/stats", authMw(http.HandlerFunc(h.GetMyStats)))
}

func (h *StatsHandler) GetMyStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var (
		authResp *pbauth.GetUserStatsResponse
		nbResp   *pbnotebook.GetUserNotebookStatsResponse
		storResp *pbstorage.GetStorageStatsResponse
		wg       sync.WaitGroup
	)

	wg.Add(3) //nolint:mnd
	go func() {
		defer wg.Done()
		authResp, _ = h.authClient.GetUserStats(r.Context(), &pbauth.GetUserStatsRequest{
			UserId:       user.ID,
			ActivityDays: 30, //nolint:mnd
		})
	}()
	go func() {
		defer wg.Done()
		nbResp, _ = h.notebookClient.GetUserStats(r.Context(), &pbnotebook.GetUserNotebookStatsRequest{
			UserId:        user.ID,
			ExecutionDays: 30, //nolint:mnd
		})
	}()
	go func() {
		defer wg.Done()
		storResp, _ = h.storageClient.GetUserStorageStats(r.Context(), &pbstorage.GetUserStorageStatsRequest{
			UserId: user.ID,
		})
	}()
	wg.Wait()

	quota := map[string]any{
		"plan":               "",
		"total_time_seconds": int64(0),
		"time_limit_seconds": int64(0),
		"usage_percent":      0.0,
	}
	activity := map[string]any{
		"last_active_at": "",
		"created_at":     "",
		"daily_activity": []any{},
	}
	if authResp != nil {
		usagePercent := 0.0
		if authResp.GetTimeLimitSeconds() > 0 {
			usagePercent = float64(authResp.GetTotalTimeSeconds()) / float64(authResp.GetTimeLimitSeconds()) * 100.0
		}
		quota["plan"] = authResp.GetPlan()
		quota["total_time_seconds"] = authResp.GetTotalTimeSeconds()
		quota["time_limit_seconds"] = authResp.GetTimeLimitSeconds()
		quota["usage_percent"] = usagePercent

		if authResp.GetLastActiveAt() > 0 {
			activity["last_active_at"] = time.Unix(authResp.GetLastActiveAt(), 0).Format(time.RFC3339)
		}
		activity["created_at"] = time.Unix(authResp.GetCreatedAt(), 0).Format(time.RFC3339)

		type dauItem struct {
			Date  string `json:"date"`
			Count int64  `json:"count"`
		}
		dau := make([]dauItem, len(authResp.GetDailyActivity()))
		for i, d := range authResp.GetDailyActivity() {
			dau[i] = dauItem{Date: d.GetDate(), Count: d.GetCount()}
		}
		activity["daily_activity"] = dau
	}

	resources := map[string]any{
		"notebook_count":   int64(0),
		"block_count":      int64(0),
		"total_executions": int64(0),
		"daily_executions": []any{},
	}
	if nbResp != nil {
		resources["notebook_count"] = nbResp.GetNotebookCount()
		resources["block_count"] = nbResp.GetBlockCount()
		resources["total_executions"] = nbResp.GetTotalExecutions()

		type execDay struct {
			Date  string `json:"date"`
			Count int64  `json:"count"`
		}
		de := make([]execDay, len(nbResp.GetDailyExecutions()))
		for i, d := range nbResp.GetDailyExecutions() {
			de[i] = execDay{Date: d.GetDate(), Count: d.GetCount()}
		}
		resources["daily_executions"] = de
	}

	storage := map[string]any{
		"total_files":       int64(0),
		"total_size_bytes":  int64(0),
		"files_by_category": map[string]int64{},
		"size_by_category":  map[string]int64{},
	}
	if storResp != nil {
		storage["total_files"] = storResp.GetTotalFiles()
		storage["total_size_bytes"] = storResp.GetTotalSizeBytes()
		storage["files_by_category"] = storResp.GetFilesByCategory()
		storage["size_by_category"] = storResp.GetSizeByCategory()
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"quota":     quota,
		"activity":  activity,
		"resources": resources,
		"storage":   storage,
	})
}
