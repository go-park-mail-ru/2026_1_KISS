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
	subRepo   repository.SubscriptionViewRepository
}

func NewEventUsecase(eventRepo repository.EventRepository, userRepo repository.UserRepository, subRepo repository.SubscriptionViewRepository) *EventUsecase {
	return &EventUsecase{eventRepo: eventRepo, userRepo: userRepo, subRepo: subRepo}
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
		if err == nil {
			if user.Plan == domain.PlanFree && user.TotalTimeSeconds+60 >= freeTimeLimitSeconds {
				_ = uc.userRepo.UpdatePlan(ctx, userID, domain.PlanFreeze)
			}
			downgradeIfExpired(ctx, uc.userRepo, uc.subRepo, user, now)
		}
	}

	return nil
}

func downgradeIfExpired(ctx context.Context, userRepo repository.UserRepository, subRepo repository.SubscriptionViewRepository, user *domain.User, now time.Time) {
	if subRepo == nil {
		return
	}
	if user.Plan != domain.PlanPro && user.Plan != domain.PlanMax {
		return
	}
	sub, err := subRepo.GetActive(ctx, user.ID)
	if err != nil {
		return
	}
	if sub == nil || sub.ExpiresAt.Before(now) {
		_ = userRepo.UpdatePlan(ctx, user.ID, domain.PlanFree)
		user.Plan = domain.PlanFree
	}
}
