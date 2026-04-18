package usecase

import (
	"context"
	"encoding/json"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type EventUsecase struct {
	eventRepo repository.EventRepository
}

func NewEventUsecase(eventRepo repository.EventRepository) *EventUsecase {
	return &EventUsecase{eventRepo: eventRepo}
}

func (uc *EventUsecase) Track(ctx context.Context, userID int64, eventType string, metadataJSON string) error {
	var metadata json.RawMessage
	if metadataJSON != "" {
		metadata = json.RawMessage(metadataJSON)
	}
	event := &domain.UserEvent{
		UserID:    userID,
		EventType: eventType,
		Metadata:  metadata,
	}
	return uc.eventRepo.Create(ctx, event)
}
