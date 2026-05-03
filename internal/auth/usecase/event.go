package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type EventUsecase struct {
	eventRepo repository.EventRepository
	userRepo  repository.UserRepository
}

func NewEventUsecase(eventRepo repository.EventRepository, userRepo repository.UserRepository) *EventUsecase {
	return &EventUsecase{eventRepo: eventRepo, userRepo: userRepo}
}

func (uc *EventUsecase) Track(ctx context.Context, userID int64, eventType string, metadataJSON string) error {
	var metadata json.RawMessage
	if metadataJSON != "" {
		if !json.Valid([]byte(metadataJSON)) {
			return domain.ErrInvalidInput
		}
		metadata = json.RawMessage(metadataJSON)
	}
	event := &domain.UserEvent{
		UserID:    userID,
		EventType: eventType,
		Metadata:  metadata,
	}
	if err := uc.eventRepo.Create(ctx, event); err != nil {
		return err
	}

	if eventType == "heartbeat" {
		now := time.Now()
		_ = uc.userRepo.UpdateLastActive(ctx, userID, now)
		_ = uc.userRepo.IncrementTotalTime(ctx, userID, 60)

		user, err := uc.userRepo.GetByID(ctx, userID)
		if err == nil && user.Plan == domain.PlanFree && user.TotalTimeSeconds+60 >= freeTimeLimitSeconds {
			_ = uc.userRepo.UpdatePlan(ctx, userID, domain.PlanFreeze)
		}
	}

	return nil
}
