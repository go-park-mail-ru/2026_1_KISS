package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pbstorage "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

func TestStatsHandler_RegisterRoutes(t *testing.T) {
	h := NewStatsHandler(nil, nil, nil)
	mux := http.NewServeMux()
	identity := func(next http.Handler) http.Handler { return next }
	h.RegisterRoutes(mux, identity)
}

func TestStatsHandler_GetMyStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	storClient := mocks.NewMockStorageServiceClient(ctrl)

	authClient.EXPECT().GetUserStats(gomock.Any(), gomock.Any()).
		Return(&pbauth.GetUserStatsResponse{
			Plan:             "free",
			TotalTimeSeconds: 5400,
			TimeLimitSeconds: 10800,
			LastActiveAt:     1714500000,
			CreatedAt:        1714400000,
			DailyActivity: []*pbauth.DauEntry{
				{Date: "2026-04-01", Count: 5},
			},
		}, nil)

	nbClient.EXPECT().GetUserStats(gomock.Any(), gomock.Any()).
		Return(&pbnotebook.GetUserNotebookStatsResponse{
			NotebookCount:   12,
			BlockCount:      87,
			TotalExecutions: 342,
		}, nil)

	storClient.EXPECT().GetUserStorageStats(gomock.Any(), gomock.Any()).
		Return(&pbstorage.GetStorageStatsResponse{
			TotalFiles:      5,
			TotalSizeBytes:  1048576,
			FilesByCategory: map[string]int64{"datasets": 3},
			SizeByCategory:  map[string]int64{"datasets": 900000},
		}, nil)

	h := NewStatsHandler(authClient, nbClient, storClient)
	req := httptest.NewRequest("GET", "/api/v1/users/me/stats", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetMyStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var envelope map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	resp, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatal("missing data in response")
	}

	quota, ok := resp["quota"].(map[string]any)
	if !ok {
		t.Fatal("missing quota in response")
	}
	if quota["plan"] != "free" {
		t.Errorf("expected plan free, got %v", quota["plan"])
	}

	resources, ok := resp["resources"].(map[string]any)
	if !ok {
		t.Fatal("missing resources in response")
	}
	if resources["notebook_count"] != float64(12) {
		t.Errorf("expected notebook_count 12, got %v", resources["notebook_count"])
	}
}

func TestStatsHandler_GetMyStats_Unauthorized(t *testing.T) {
	h := NewStatsHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/users/me/stats", nil)
	rec := httptest.NewRecorder()

	h.GetMyStats(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestStatsHandler_GetMyStats_PartialFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	storClient := mocks.NewMockStorageServiceClient(ctrl)

	authClient.EXPECT().GetUserStats(gomock.Any(), gomock.Any()).
		Return(&pbauth.GetUserStatsResponse{
			Plan:             "pro",
			TotalTimeSeconds: 50000,
		}, nil)

	nbClient.EXPECT().GetUserStats(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("service unavailable"))

	storClient.EXPECT().GetUserStorageStats(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("service unavailable"))

	h := NewStatsHandler(authClient, nbClient, storClient)
	req := httptest.NewRequest("GET", "/api/v1/users/me/stats", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetMyStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 even on partial failure, got %d", rec.Code)
	}

	var envelope map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	resp, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatal("missing data in response")
	}

	resources, ok := resp["resources"].(map[string]any)
	if !ok {
		t.Fatal("missing resources in response")
	}
	if resources["notebook_count"] != float64(0) {
		t.Errorf("expected 0 for failed service, got %v", resources["notebook_count"])
	}
}
