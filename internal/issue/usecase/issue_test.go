package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/issue/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestIssueService_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, issue *domain.Issue) (int64, error) {
			if issue.Category != domain.CategoryBug {
				t.Errorf("want CategoryBug, got %s", issue.Category)
			}
			if issue.Status != domain.IssueStatusOpen {
				t.Errorf("want StatusOpen, got %s", issue.Status)
			}
			if issue.Content != "Test content" {
				t.Errorf("want Test content, got %s", issue.Content)
			}
			return 1, nil
		})

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		Category: domain.CategoryBug,
		Content:  "Test content",
		UserID:   100,
	}
	id, err := uc.Create(context.Background(), issue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1 {
		t.Errorf("want ID=1, got %d", id)
	}
}

func TestIssueService_Create_DefaultStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, issue *domain.Issue) (int64, error) {
			if issue.Status != domain.IssueStatusOpen {
				t.Errorf("want default StatusOpen, got %s", issue.Status)
			}
			return 1, nil
		})

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		Category: domain.CategoryIdea,
		Content:  "My idea",
		UserID:   100,
	}
	_, err := uc.Create(context.Background(), issue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueService_Create_EmptyContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		Category: domain.CategoryBug,
		Content:  "",
		UserID:   100,
	}
	_, err := uc.Create(context.Background(), issue)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestIssueService_Create_InvalidCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		Category: "INVALID",
		Content:  "Test",
		UserID:   100,
	}
	_, err := uc.Create(context.Background(), issue)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestIssueService_Create_ContentTooLong(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)

	uc := usecase.NewIssueService(issueRepo)
	longContent := string(make([]byte, 5001))
	issue := &domain.Issue{
		Category: domain.CategoryBug,
		Content:  longContent,
		UserID:   100,
	}
	_, err := uc.Create(context.Background(), issue)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestIssueService_Create_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(int64(0), errors.New("db error"))

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		Category: domain.CategoryBug,
		Content:  "Test",
		UserID:   100,
	}
	_, err := uc.Create(context.Background(), issue)
	if err == nil {
		t.Error("expected error")
	}
}

func TestIssueService_GetByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Issue{
			ID:       42,
			Category: domain.CategoryBug,
			Status:   domain.IssueStatusOpen,
			Content:  "Test issue",
			UserID:   100,
		}, nil)

	uc := usecase.NewIssueService(issueRepo)
	issue, err := uc.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.ID != 42 {
		t.Errorf("want ID=42, got %d", issue.ID)
	}
	if issue.Category != domain.CategoryBug {
		t.Errorf("want CategoryBug, got %s", issue.Category)
	}
}

func TestIssueService_GetByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(999)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.NewIssueService(issueRepo)
	_, err := uc.GetByID(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestIssueService_GetByID_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(nil, errors.New("db error"))

	uc := usecase.NewIssueService(issueRepo)
	_, err := uc.GetByID(context.Background(), 42)
	if err == nil {
		t.Error("expected error")
	}
}

func TestIssueService_GetAll_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issues := []domain.Issue{
		{ID: 1, Category: domain.CategoryBug, Status: domain.IssueStatusOpen, Content: "Issue 1", UserID: 100},
		{ID: 2, Category: domain.CategoryIdea, Status: domain.IssueStatusInWork, Content: "Issue 2", UserID: 100},
	}
	issueRepo.EXPECT().GetAll(gomock.Any(), 20, 0, (*domain.IssueFilter)(nil)).
		Return(issues, nil)

	uc := usecase.NewIssueService(issueRepo)
	result, err := uc.GetAll(context.Background(), 0, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("want 2 issues, got %d", len(result))
	}
}

func TestIssueService_GetAll_WithFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	filter := &domain.IssueFilter{
		Category: domain.CategoryBug,
		Status:   domain.IssueStatusOpen,
		UserID:   100,
	}
	issueRepo.EXPECT().GetAll(gomock.Any(), 10, 0, filter).
		Return([]domain.Issue{
			{ID: 1, Category: domain.CategoryBug, Status: domain.IssueStatusOpen, Content: "Bug report", UserID: 100},
		}, nil)

	uc := usecase.NewIssueService(issueRepo)
	result, err := uc.GetAll(context.Background(), 10, 0, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("want 1 issue, got %d", len(result))
	}
}

func TestIssueService_GetAll_NormalizesLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetAll(gomock.Any(), 20, 0, (*domain.IssueFilter)(nil)).
		Return([]domain.Issue{}, nil)

	uc := usecase.NewIssueService(issueRepo)
	_, err := uc.GetAll(context.Background(), -5, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueService_GetAll_NormalizesOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetAll(gomock.Any(), 20, 0, (*domain.IssueFilter)(nil)).
		Return([]domain.Issue{}, nil)

	uc := usecase.NewIssueService(issueRepo)
	_, err := uc.GetAll(context.Background(), 20, -10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueService_GetAll_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetAll(gomock.Any(), 20, 0, (*domain.IssueFilter)(nil)).
		Return(nil, errors.New("db error"))

	uc := usecase.NewIssueService(issueRepo)
	_, err := uc.GetAll(context.Background(), 0, 0, nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestIssueService_Update_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Issue{
			ID:       42,
			Category: domain.CategoryBug,
			Status:   domain.IssueStatusOpen,
			Content:  "Old content",
			UserID:   100,
		}, nil)
	issueRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, issue *domain.Issue) error {
			if issue.Content != "Updated content" {
				t.Errorf("want Updated content, got %s", issue.Content)
			}
			if issue.Category != domain.CategoryIdea {
				t.Errorf("want CategoryIdea, got %s", issue.Category)
			}
			return nil
		})

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		ID:       42,
		Category: domain.CategoryIdea,
		Status:   domain.IssueStatusInWork,
		Content:  "Updated content",
		UserID:   100,
	}
	err := uc.Update(context.Background(), issue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueService_Update_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Issue{
			ID:     42,
			UserID: 200, // Different user
		}, nil)

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		ID:     42,
		UserID: 100, // Trying to update as different user
	}
	err := uc.Update(context.Background(), issue)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestIssueService_Update_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		ID:     42,
		UserID: 100,
	}
	err := uc.Update(context.Background(), issue)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestIssueService_Update_EmptyContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Issue{
			ID:     42,
			UserID: 100,
		}, nil)

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		ID:      42,
		Content: "",
		UserID:  100,
	}
	err := uc.Update(context.Background(), issue)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestIssueService_Update_InvalidCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Issue{
			ID:     42,
			UserID: 100,
		}, nil)

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		ID:       42,
		Category: "INVALID",
		Content:  "Test",
		UserID:   100,
	}
	err := uc.Update(context.Background(), issue)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestIssueService_Update_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Issue{
			ID:       42,
			Category: domain.CategoryBug,
			Content:  "Test",
			UserID:   100,
		}, nil)
	issueRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
		Return(errors.New("db error"))

	uc := usecase.NewIssueService(issueRepo)
	issue := &domain.Issue{
		ID:       42,
		Category: domain.CategoryBug,
		Content:  "Updated",
		UserID:   100,
	}
	err := uc.Update(context.Background(), issue)
	if err == nil {
		t.Error("expected error")
	}
}

func TestIssueService_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().Delete(gomock.Any(), int64(42)).
		Return(nil)

	uc := usecase.NewIssueService(issueRepo)
	err := uc.Delete(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueService_Delete_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().Delete(gomock.Any(), int64(999)).
		Return(domain.ErrNotFound)

	uc := usecase.NewIssueService(issueRepo)
	err := uc.Delete(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestIssueService_Delete_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueRepo.EXPECT().Delete(gomock.Any(), int64(42)).
		Return(errors.New("db error"))

	uc := usecase.NewIssueService(issueRepo)
	err := uc.Delete(context.Background(), 42)
	if err == nil {
		t.Error("expected error")
	}
}
