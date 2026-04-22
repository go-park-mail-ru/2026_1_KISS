package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

func TestEventHandler_Track_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().TrackEvent(gomock.Any(), gomock.Any()).
		Return(&pb.TrackEventResponse{}, nil)

	h := NewEventHandler(client)
	body, _ := json.Marshal(trackEventRequest{EventType: "page_view", Metadata: `{"page":"/home"}`})
	req := httptest.NewRequest("POST", "/api/v1/events/track", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Track(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestEventHandler_Track_Unauthorized(t *testing.T) {
	h := NewEventHandler(nil)
	req := httptest.NewRequest("POST", "/api/v1/events/track", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	h.Track(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestEventHandler_Track_InvalidBody(t *testing.T) {
	h := NewEventHandler(nil)
	req := httptest.NewRequest("POST", "/api/v1/events/track", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Track(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestEventHandler_Track_EmptyEventType(t *testing.T) {
	h := NewEventHandler(nil)
	body, _ := json.Marshal(trackEventRequest{EventType: ""})
	req := httptest.NewRequest("POST", "/api/v1/events/track", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Track(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestEventHandler_Track_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().TrackEvent(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "internal"))

	h := NewEventHandler(client)
	body, _ := json.Marshal(trackEventRequest{EventType: "click"})
	req := httptest.NewRequest("POST", "/api/v1/events/track", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Track(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}
