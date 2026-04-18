package usecase

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

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

func (uc *AdminUsecase) ListUsers(ctx context.Context, adminUserID int64, limit, offset int, search string) ([]domain.User, int, error) {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return nil, 0, err
	}
	return uc.userRepo.ListAll(ctx, limit, offset, search)
}

func (uc *AdminUsecase) SetBan(ctx context.Context, adminUserID, targetUserID int64, ban bool) error {
	if err := uc.checkAdmin(ctx, adminUserID); err != nil {
		return err
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

	return &PlatformStats{
		DAU: dau,
		MAU: mau,
	}, nil
}
