package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestStatsUsecase_GetUserStats_FreePlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	now := time.Now()
	user := &domain.User{
		ID:               1,
		Plan:             domain.PlanFree,
		TotalTimeSeconds: 5400,
		LastActiveAt:     &now,
		CreatedAt:        now.Add(-24 * time.Hour),
	}

	activity := []domain.DayCount{
		{Date: now, Count: 10},
	}

	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	eventRepo.EXPECT().CountUserActivityByDay(gomock.Any(), int64(1), gomock.Any()).Return(activity, nil)

	uc := NewStatsUsecase(userRepo, eventRepo)
	stats, err := uc.GetUserStats(context.Background(), 1, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Plan != domain.PlanFree {
		t.Errorf("expected plan %s, got %s", domain.PlanFree, stats.Plan)
	}
	if stats.TimeLimitSeconds != freeTimeLimitSeconds {
		t.Errorf("expected time limit %d, got %d", freeTimeLimitSeconds, stats.TimeLimitSeconds)
	}
	if stats.TotalTimeSeconds != 5400 {
		t.Errorf("expected total time 5400, got %d", stats.TotalTimeSeconds)
	}
	if len(stats.DailyActivity) != 1 {
		t.Errorf("expected 1 activity entry, got %d", len(stats.DailyActivity))
	}
}

func TestStatsUsecase_GetUserStats_ProPlan_Unlimited(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	user := &domain.User{
		ID:               2,
		Plan:             domain.PlanPro,
		TotalTimeSeconds: 50000,
		CreatedAt:        time.Now(),
	}

	userRepo.EXPECT().GetByID(gomock.Any(), int64(2)).Return(user, nil)
	eventRepo.EXPECT().CountUserActivityByDay(gomock.Any(), int64(2), gomock.Any()).Return(nil, nil)

	uc := NewStatsUsecase(userRepo, eventRepo)
	stats, err := uc.GetUserStats(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TimeLimitSeconds != 0 {
		t.Errorf("expected unlimited (0), got %d", stats.TimeLimitSeconds)
	}
}

func TestStatsUsecase_GetUserStats_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	userRepo.EXPECT().GetByID(gomock.Any(), int64(99)).Return(nil, domain.ErrNotFound)

	uc := NewStatsUsecase(userRepo, eventRepo)
	_, err := uc.GetUserStats(context.Background(), 99, 30)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
