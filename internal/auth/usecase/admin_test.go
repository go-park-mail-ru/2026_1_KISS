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
		userRepo.EXPECT().ListAll(gomock.Any(), 10, 0, "", (*bool)(nil)).Return(users, 1, nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	result, total, err := uc.ListUsers(context.Background(), 1, 10, 0, "", nil)

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
	_, _, err := uc.ListUsers(context.Background(), 1, 10, 0, "", nil)

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
	targetUser := &domain.User{ID: 2, Username: "user", Plan: domain.PlanFreeze}

	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		userRepo.EXPECT().GetByID(gomock.Any(), int64(2)).Return(targetUser, nil),
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
		userRepo.EXPECT().CountAll(gomock.Any()).Return(int64(42), nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	stats, err := uc.GetStats(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.DAU != 100 || stats.MAU != 500 {
		t.Fatalf("unexpected stats: DAU=%d, MAU=%d", stats.DAU, stats.MAU)
	}
	if stats.TotalUsers != 42 {
		t.Fatalf("expected TotalUsers=42, got %d", stats.TotalUsers)
	}
}

func TestEventUsecase_Track_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventRepo := mocks.NewMockEventRepository(ctrl)
	eventRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	userRepo := mocks.NewMockUserRepository(ctrl)
	uc := NewEventUsecase(eventRepo, userRepo, nil)
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

	userRepo := mocks.NewMockUserRepository(ctrl)
	uc := NewEventUsecase(eventRepo, userRepo, nil)
	err := uc.Track(context.Background(), 1, "logout", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminUsecase_AdminUpdateUser_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	updatedUser := &domain.User{ID: 2, Username: "newname", Email: "new@mail.com"}

	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		userRepo.EXPECT().AdminUpdateUser(gomock.Any(), int64(2), "newname", "new@mail.com").Return(nil),
		userRepo.EXPECT().GetByID(gomock.Any(), int64(2)).Return(updatedUser, nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	user, err := uc.AdminUpdateUser(context.Background(), 1, 2, "newname", "new@mail.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "newname" {
		t.Fatalf("expected newname, got %s", user.Username)
	}
}

func TestAdminUsecase_AdminSetPlan_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		userRepo.EXPECT().UpdatePlan(gomock.Any(), int64(2), "pro").Return(nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	err := uc.AdminSetPlan(context.Background(), 1, 2, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminUsecase_AdminSetPlan_InvalidPlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil)

	uc := NewAdminUsecase(userRepo, eventRepo)
	err := uc.AdminSetPlan(context.Background(), 1, 2, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid plan")
	}
}

func TestAdminUsecase_SetBan_OnlyFreezeAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	targetUser := &domain.User{ID: 2, Plan: domain.PlanFree}

	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		userRepo.EXPECT().GetByID(gomock.Any(), int64(2)).Return(targetUser, nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	err := uc.SetBan(context.Background(), 1, 2, true)
	if err == nil {
		t.Fatal("expected error when banning non-freeze user")
	}
}

func TestEventUsecase_Track_Heartbeat_UpdatesActivity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventRepo := mocks.NewMockEventRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)

	eventRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	userRepo.EXPECT().UpdateLastActive(gomock.Any(), int64(1), gomock.Any()).Return(nil)
	userRepo.EXPECT().IncrementTotalTime(gomock.Any(), int64(1), int64(60)).Return(nil)
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.User{ID: 1, Plan: domain.PlanPro, TotalTimeSeconds: 100}, nil)

	uc := NewEventUsecase(eventRepo, userRepo, nil)
	err := uc.Track(context.Background(), 1, "heartbeat", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminUsecase_AdminResetPassword_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	gomock.InOrder(
		userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil),
		userRepo.EXPECT().UpdatePassword(gomock.Any(), int64(2), gomock.Any()).Return(nil),
	)

	uc := NewAdminUsecase(userRepo, eventRepo)
	err := uc.AdminResetPassword(context.Background(), 1, 2, "newpassword123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminUsecase_AdminResetPassword_TooShort(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil)

	uc := NewAdminUsecase(userRepo, eventRepo)
	err := uc.AdminResetPassword(context.Background(), 1, 2, "short")
	if err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestAdminUsecase_AdminUpdateUser_EmptyFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil)

	uc := NewAdminUsecase(userRepo, eventRepo)
	_, err := uc.AdminUpdateUser(context.Background(), 1, 2, "", "")
	if err == nil {
		t.Fatal("expected error for empty fields")
	}
}

func TestEventUsecase_Track_Heartbeat_AutoFreeze(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventRepo := mocks.NewMockEventRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)

	eventRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	userRepo.EXPECT().UpdateLastActive(gomock.Any(), int64(1), gomock.Any()).Return(nil)
	userRepo.EXPECT().IncrementTotalTime(gomock.Any(), int64(1), int64(60)).Return(nil)
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.User{
		ID: 1, Plan: domain.PlanFree, TotalTimeSeconds: freeTimeLimitSeconds - 30,
	}, nil)
	userRepo.EXPECT().UpdatePlan(gomock.Any(), int64(1), domain.PlanFreeze).Return(nil)

	uc := NewEventUsecase(eventRepo, userRepo, nil)
	err := uc.Track(context.Background(), 1, "heartbeat", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminUsecase_GetActivityStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)

	adminUser := &domain.User{ID: 1, IsAdmin: true}
	userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(adminUser, nil)
	eventRepo.EXPECT().CountActiveUsersByDay(gomock.Any(), gomock.Any()).Return([]domain.DayCount{{Count: 5}}, nil)
	eventRepo.EXPECT().CountActiveUsersByMonth(gomock.Any(), gomock.Any()).Return([]domain.MonthCount{{Count: 50}}, nil)

	uc := NewAdminUsecase(userRepo, eventRepo)
	dau, mau, err := uc.GetActivityStats(context.Background(), 1, 30, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dau) != 1 || dau[0].Count != 5 {
		t.Fatalf("unexpected dau: %v", dau)
	}
	if len(mau) != 1 || mau[0].Count != 50 {
		t.Fatalf("unexpected mau: %v", mau)
	}
}
