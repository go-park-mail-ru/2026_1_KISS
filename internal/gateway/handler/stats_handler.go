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

type statsResponse struct {
	Quota     quotaStats    `json:"quota"`
	Activity  activityStats `json:"activity"`
	Resources resourceStats `json:"resources"`
	Storage   storageStats  `json:"storage"`
}

type quotaStats struct {
	Plan             string  `json:"plan"`
	TotalTimeSeconds int64   `json:"total_time_seconds"`
	TimeLimitSeconds int64   `json:"time_limit_seconds"`
	UsagePercent     float64 `json:"usage_percent"`
}

type activityStats struct {
	LastActiveAt  string    `json:"last_active_at"`
	CreatedAt     string    `json:"created_at"`
	DailyActivity []dauItem `json:"daily_activity"`
}

type dauItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type resourceStats struct {
	NotebookCount   int64      `json:"notebook_count"`
	BlockCount      int64      `json:"block_count"`
	TotalExecutions int64      `json:"total_executions"`
	DailyExecutions []execItem `json:"daily_executions"`
}

type execItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type storageStats struct {
	TotalFiles      int64            `json:"total_files"`
	TotalSizeBytes  int64            `json:"total_size_bytes"`
	FilesByCategory map[string]int64 `json:"files_by_category"`
	SizeByCategory  map[string]int64 `json:"size_by_category"`
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

	resp := statsResponse{
		Quota: quotaStats{
			Plan:             "",
			TotalTimeSeconds: 0,
			TimeLimitSeconds: 0,
			UsagePercent:     0,
		},
		Activity: activityStats{
			LastActiveAt:  "",
			CreatedAt:     "",
			DailyActivity: []dauItem{},
		},
		Resources: resourceStats{
			NotebookCount:   0,
			BlockCount:      0,
			TotalExecutions: 0,
			DailyExecutions: []execItem{},
		},
		Storage: storageStats{
			TotalFiles:      0,
			TotalSizeBytes:  0,
			FilesByCategory: map[string]int64{},
			SizeByCategory:  map[string]int64{},
		},
	}

	if authResp != nil {
		usagePercent := 0.0
		if authResp.GetTimeLimitSeconds() > 0 {
			usagePercent = float64(authResp.GetTotalTimeSeconds()) / float64(authResp.GetTimeLimitSeconds()) * 100.0
		}
		resp.Quota.Plan = authResp.GetPlan()
		resp.Quota.TotalTimeSeconds = authResp.GetTotalTimeSeconds()
		resp.Quota.TimeLimitSeconds = authResp.GetTimeLimitSeconds()
		resp.Quota.UsagePercent = usagePercent

		if authResp.GetLastActiveAt() > 0 {
			resp.Activity.LastActiveAt = time.Unix(authResp.GetLastActiveAt(), 0).Format(time.RFC3339)
		}
		resp.Activity.CreatedAt = time.Unix(authResp.GetCreatedAt(), 0).Format(time.RFC3339)

		dau := make([]dauItem, len(authResp.GetDailyActivity()))
		for i, d := range authResp.GetDailyActivity() {
			dau[i] = dauItem{Date: d.GetDate(), Count: d.GetCount()}
		}
		resp.Activity.DailyActivity = dau
	}

	if nbResp != nil {
		resp.Resources.NotebookCount = nbResp.GetNotebookCount()
		resp.Resources.BlockCount = nbResp.GetBlockCount()
		resp.Resources.TotalExecutions = nbResp.GetTotalExecutions()

		de := make([]execItem, len(nbResp.GetDailyExecutions()))
		for i, d := range nbResp.GetDailyExecutions() {
			de[i] = execItem{Date: d.GetDate(), Count: d.GetCount()}
		}
		resp.Resources.DailyExecutions = de
	}

	if storResp != nil {
		resp.Storage.TotalFiles = storResp.GetTotalFiles()
		resp.Storage.TotalSizeBytes = storResp.GetTotalSizeBytes()
		resp.Storage.FilesByCategory = storResp.GetFilesByCategory()
		resp.Storage.SizeByCategory = storResp.GetSizeByCategory()
	}

	httputil.JSON(w, http.StatusOK, resp)
}
