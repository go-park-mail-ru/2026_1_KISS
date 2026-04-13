package health_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/health"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestHealth_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pinger := mocks.NewMockPinger(ctrl)
	pinger.EXPECT().PingContext(gomock.Any()).Return(nil)

	h := health.NewWithPinger(pinger)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	h.Health(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHealth_DBDown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pinger := mocks.NewMockPinger(ctrl)
	pinger.EXPECT().PingContext(gomock.Any()).Return(errors.New("connection refused"))

	h := health.NewWithPinger(pinger)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	h.Health(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("want 503, got %d", rec.Code)
	}
}
