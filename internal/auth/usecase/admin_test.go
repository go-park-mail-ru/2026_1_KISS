package usecase

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestAdminUsecase_ListUsers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, Username: "admin", IsAdmin: true}
	users := []domain.User{{ID: 2, Username: "user1"}}

	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		userRepo.EXPECT().ListAll(gomock.Any(), 10, 0, "").Return(users, 1, nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	result, total, err := uc.ListUsers(context.Background(), 1, 10, 0, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(result) != 1 {
		t.Fatalf("expected 1 user, got %d", len(result))
	}
}

func TestAdminUsecase_ListUsers_NotAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	nonAdminUser := &domain.User{ID: 1, Username: "user", IsAdmin: false}
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nonAdminUser, nil)

	uc := NewAdminUsecase(userRepo, eventRepo)
	_, _, err := uc.ListUsers(context.Background(), 1, 10, 0, "")

	if err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAdminUsecase_SetBan_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, Username: "admin", IsAdmin: true}

	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		userRepo.EXPECT().SetBanned(gomock.Any(), int64(2), true).Return(nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	err := uc.SetBan(context.Background(), 1, 2, true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminUsecase_GetStats_NotAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	nonAdminUser := &domain.User{ID: 1, Username: "user", IsAdmin: false}
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nonAdminUser, nil)

	uc := NewAdminUsecase(userRepo, eventRepo)
	_, err := uc.GetStats(context.Background(), 1)

	if err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAdminUsecase_GetStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, Username: "admin", IsAdmin: true}

	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		eventRepo.EXPECT().CountActiveUsers(gomock.Any(), gomock.Any()).Return(int64(100), nil),
		eventRepo.EXPECT().CountActiveUsers(gomock.Any(), gomock.Any()).Return(int64(500), nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	stats, err := uc.GetStats(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.DAU != 100 || stats.MAU != 500 {
		t.Fatalf("unexpected stats: DAU=%d, MAU=%d", stats.DAU, stats.MAU)
	}
}

func TestEventUsecase_Track_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventRepo := mocks.NewMockEventRepository(ctrl)
	eventRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	uc := NewEventUsecase(eventRepo)
	err := uc.Track(context.Background(), 1, "login", "{}")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventUsecase_Track_EmptyMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventRepo := mocks.NewMockEventRepository(ctrl)
	eventRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	uc := NewEventUsecase(eventRepo)
	err := uc.Track(context.Background(), 1, "logout", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
