package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

const freeTimeLimitSeconds = 3 * 3600

type AdminUsecase struct {
	userRepo  repository.UserRepository
	eventRepo repository.EventRepository
}

func NewAdminUsecase(userRepo repository.UserRepository, eventRepo repository.EventRepository) *AdminUsecase {
	return &AdminUsecase{userRepo: userRepo, eventRepo: eventRepo}
}

func (uc *AdminUsecase) checkAdmin(ctx context.Context, adminUserID int64) error {
	user, err := uc.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return domain.ErrForbidden
	}
	return nil
}

func (uc *AdminUsecase) ListUsers(ctx context.Context, adminUserID int64, limit, offset int, search string, verified *bool) ([]domain.User, int, error) {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return nil, 0, err
	}
	return uc.userRepo.ListAll(ctx, limit, offset, search, verified)
}

func (uc *AdminUsecase) SetBan(ctx context.Context, adminUserID, targetUserID int64, ban bool) error {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return err
	}
	if ban {
		target, err := uc.userRepo.GetByID(ctx, targetUserID)
		if err != nil {
			return err
		}
		if target.Plan != domain.PlanFreeze {
			return fmt.Errorf("%w: can only ban users with freeze plan", domain.ErrInvalidInput)
		}
	}
	return uc.userRepo.SetBanned(ctx, targetUserID, ban)
}

type PlatformStats struct {
	TotalUsers    int64
	TotalSessions int64
	DAU           int64
	MAU           int64
}

func (uc *AdminUsecase) GetStats(ctx context.Context, adminUserID int64) (*PlatformStats, error) {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return nil, err
	}

	now := time.Now()
	dau, err := uc.eventRepo.CountActiveUsers(ctx, now.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	mau, err := uc.eventRepo.CountActiveUsers(ctx, now.AddDate(0, -1, 0))
	if err != nil {
		return nil, err
	}
	totalUsers, err := uc.userRepo.CountAll(ctx)
	if err != nil {
		return nil, err
	}

	return &PlatformStats{
		TotalUsers: totalUsers,
		DAU:        dau,
		MAU:        mau,
	}, nil
}

func (uc *AdminUsecase) AdminUpdateUser(ctx context.Context, adminUserID, targetUserID int64, username, email string) (*domain.User, error) {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return nil, err
	}
	if username == "" || email == "" {
		return nil, fmt.Errorf("%w: username and email are required", domain.ErrInvalidInput)
	}
	if err := uc.userRepo.AdminUpdateUser(ctx, targetUserID, username, email); err != nil {
		return nil, err
	}
	return uc.userRepo.GetByID(ctx, targetUserID)
}

func (uc *AdminUsecase) AdminResetPassword(ctx context.Context, adminUserID, targetUserID int64, newPassword string) error {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return err
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters", domain.ErrInvalidInput)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return uc.userRepo.UpdatePassword(ctx, targetUserID, string(hash))
}

func (uc *AdminUsecase) AdminSetPlan(ctx context.Context, adminUserID, targetUserID int64, plan string) error {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return err
	}
	if !domain.ValidPlans[plan] {
		return fmt.Errorf("%w: invalid plan: %s", domain.ErrInvalidInput, plan)
	}
	return uc.userRepo.UpdatePlan(ctx, targetUserID, plan)
}

func (uc *AdminUsecase) SetUserPlanInternal(ctx context.Context, targetUserID int64, plan string) error {
	if !domain.ValidPlans[plan] {
		return fmt.Errorf("%w: invalid plan: %s", domain.ErrInvalidInput, plan)
	}
	return uc.userRepo.UpdatePlan(ctx, targetUserID, plan)
}

func (uc *AdminUsecase) GetActivityStats(ctx context.Context, adminUserID int64, dauDays, mauMonths int) ([]domain.DayCount, []domain.MonthCount, error) {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return nil, nil, err
	}
	now := time.Now()
	dau, err := uc.eventRepo.CountActiveUsersByDay(ctx, now.AddDate(0, 0, -dauDays))
	if err != nil {
		return nil, nil, err
	}
	mau, err := uc.eventRepo.CountActiveUsersByMonth(ctx, now.AddDate(0, -mauMonths, 0))
	if err != nil {
		return nil, nil, err
	}
	return dau, mau, nil
}
