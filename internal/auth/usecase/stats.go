package usecase

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type UserQuotaStats struct {
	Plan             string
	TotalTimeSeconds int64
	TimeLimitSeconds int64
	LastActiveAt     *time.Time
	CreatedAt        time.Time
	DailyActivity    []domain.DayCount
}

type StatsUsecase struct {
	userRepo  repository.UserRepository
	eventRepo repository.EventRepository
}

func NewStatsUsecase(userRepo repository.UserRepository, eventRepo repository.EventRepository) *StatsUsecase {
	return &StatsUsecase{userRepo: userRepo, eventRepo: eventRepo}
}

func (uc *StatsUsecase) GetUserStats(ctx context.Context, userID int64, activityDays int) (*UserQuotaStats, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if activityDays <= 0 {
		activityDays = 30
	}

	var timeLimit int64
	if user.Plan == domain.PlanFree || user.Plan == domain.PlanFreeze {
		timeLimit = freeTimeLimitSeconds
	}

	since := time.Now().AddDate(0, 0, -activityDays)
	activity, err := uc.eventRepo.CountUserActivityByDay(ctx, userID, since)
	if err != nil {
		return nil, err
	}

	return &UserQuotaStats{
		Plan:             user.Plan,
		TotalTimeSeconds: user.TotalTimeSeconds,
		TimeLimitSeconds: timeLimit,
		LastActiveAt:     user.LastActiveAt,
		CreatedAt:        user.CreatedAt,
		DailyActivity:    activity,
	}, nil
}
