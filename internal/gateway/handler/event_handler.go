package handler

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

type EventHandler struct {
	client pb.AuthServiceClient
}

func NewEventHandler(client pb.AuthServiceClient) *EventHandler {
	return &EventHandler{client: client}
}

func (h *EventHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("POST /api/v1/events/track", authMw(http.HandlerFunc(h.Track)))
}

type trackEventRequest struct {
	EventType string `json:"event_type"`
	Metadata  string `json:"metadata"`
}

func (h *EventHandler) Track(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req trackEventRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.EventType == "" {
		httputil.Error(w, http.StatusBadRequest, "event_type is required")
		return
	}

	_, err := h.client.TrackEvent(r.Context(), &pb.TrackEventRequest{
		UserId:       user.ID,
		EventType:    req.EventType,
		MetadataJson: req.Metadata,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}
