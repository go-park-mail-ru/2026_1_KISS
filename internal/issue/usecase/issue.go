package usecase

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/issue/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/sanitize"
)

type IssueService interface {
	GetByID(ctx context.Context, id int64) (*domain.Issue, error)
	GetAll(ctx context.Context, limit, offset int, filter *domain.IssueFilter) ([]domain.Issue, error)
	Create(ctx context.Context, issue *domain.Issue) (int64, error)
	Update(ctx context.Context, issue *domain.Issue) error
	Delete(ctx context.Context, id int64, userID int64) error
	AdminUpdateStatus(ctx context.Context, id int64, status domain.IssueStatus) error
	GetStats(ctx context.Context) (*domain.IssueStats, error)
}

type IssueMessageService interface {
	AddMessage(ctx context.Context, msg *domain.IssueMessage) (int64, error)
	GetByIssueID(ctx context.Context, issueID int64) ([]domain.IssueMessage, error)
}

type issueService struct {
	issueRepo repository.IssueRepository
}

func NewIssueService(ir repository.IssueRepository) IssueService {
	return &issueService{issueRepo: ir}
}

type issueMessageService struct {
	msgRepo repository.IssueMessageRepository
}

func NewIssueMessageService(mr repository.IssueMessageRepository) IssueMessageService {
	return &issueMessageService{msgRepo: mr}
}

func (s *issueService) GetByID(ctx context.Context, id int64) (*domain.Issue, error) {
	logger.Info(ctx, "usecase.issue.GetByID", "issue_id", id)

	issue, err := s.issueRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			logger.Error(ctx, "usecase.issue.GetByID", "error", domain.ErrNotFound)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "usecase.issue.GetByID", "error", err)
		return nil, err
	}

	logger.Info(ctx, "usecase.issue.GetByID", "issue_id", issue.ID)
	return issue, nil
}

func (s *issueService) GetAll(ctx context.Context, limit, offset int, filter *domain.IssueFilter) ([]domain.Issue, error) {
	logger.Info(ctx, "usecase.issue.GetAll", "limit", limit, "offset", offset)

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	issues, err := s.issueRepo.GetAll(ctx, limit, offset, filter)
	if err != nil {
		logger.Error(ctx, "usecase.issue.GetAll", "error", err)
		return nil, err
	}

	logger.Info(ctx, "usecase.issue.GetAll", "count", len(issues))
	return issues, nil
}

func (s *issueService) Create(ctx context.Context, issue *domain.Issue) (int64, error) {
	logger.Info(ctx, "usecase.issue.Create", "user_id", issue.UserID, "category", issue.Category)

	if err := s.validateIssue(issue); err != nil {
		logger.Error(ctx, "usecase.issue.Create", "error", err)
		return 0, err
	}

	issue.Content = sanitize.EscapeHTML(issue.Content)
	if issue.Content == "" {
		logger.Error(ctx, "usecase.issue.Create", "error", domain.ErrInvalidInput)
		return 0, domain.ErrInvalidInput
	}

	if issue.Status == "" {
		issue.Status = domain.IssueStatusOpen
	}

	id, err := s.issueRepo.Create(ctx, issue)
	if err != nil {
		logger.Error(ctx, "usecase.issue.Create", "error", err)
		return 0, err
	}

	logger.Info(ctx, "usecase.issue.Create", "issue_id", id)
	return id, nil
}

func (s *issueService) Update(ctx context.Context, issue *domain.Issue) error {
	logger.Info(ctx, "usecase.issue.Update", "issue_id", issue.ID, "user_id", issue.UserID)

	existing, err := s.issueRepo.GetByID(ctx, issue.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return err
	}

	if existing.UserID != issue.UserID {
		logger.Error(ctx, "usecase.issue.Update", "error", domain.ErrForbidden)
		return domain.ErrForbidden
	}

	if err := s.validateIssue(issue); err != nil {
		logger.Error(ctx, "usecase.issue.Update", "error", err)
		return err
	}

	issue.Content = sanitize.EscapeHTML(issue.Content)
	if issue.Content == "" {
		return domain.ErrInvalidInput
	}

	if err := s.issueRepo.Update(ctx, issue); err != nil {
		logger.Error(ctx, "usecase.issue.Update", "error", err)
		return err
	}

	logger.Info(ctx, "usecase.issue.Update", "issue_id", issue.ID)
	return nil
}

func (s *issueService) Delete(ctx context.Context, id int64, userID int64) error {
	logger.Info(ctx, "usecase.issue.Delete", "issue_id", id, "user_id", userID)

	if err := s.issueRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		logger.Error(ctx, "usecase.issue.Delete", "error", err)
		return err
	}

	logger.Info(ctx, "usecase.issue.Delete", "issue_id", id, "status", "ok")
	return nil
}

func (s *issueService) AdminUpdateStatus(ctx context.Context, id int64, status domain.IssueStatus) error {
	logger.Info(ctx, "usecase.issue.AdminUpdateStatus", "issue_id", id, "status", status)

	if !s.isValidStatus(status) {
		return domain.ErrInvalidInput
	}

	if err := s.issueRepo.AdminUpdateStatus(ctx, id, status); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		logger.Error(ctx, "usecase.issue.AdminUpdateStatus", "error", err)
		return err
	}

	logger.Info(ctx, "usecase.issue.AdminUpdateStatus", "issue_id", id)
	return nil
}

func (s *issueService) GetStats(ctx context.Context) (*domain.IssueStats, error) {
	logger.Info(ctx, "usecase.issue.GetStats")

	stats, err := s.issueRepo.GetStats(ctx)
	if err != nil {
		logger.Error(ctx, "usecase.issue.GetStats", "error", err)
		return nil, err
	}

	return stats, nil
}

func (s *issueMessageService) AddMessage(ctx context.Context, msg *domain.IssueMessage) (int64, error) {
	logger.Info(ctx, "usecase.issue_message.AddMessage", "issue_id", msg.IssueID, "user_id", msg.UserID)

	msg.Content = sanitize.EscapeHTML(msg.Content)
	if utf8.RuneCountInString(msg.Content) == 0 {
		return 0, domain.ErrInvalidInput
	}
	if utf8.RuneCountInString(msg.Content) > 4000 {
		return 0, domain.ErrInvalidInput
	}

	id, err := s.msgRepo.Create(ctx, msg)
	if err != nil {
		logger.Error(ctx, "usecase.issue_message.AddMessage", "error", err)
		return 0, err
	}

	logger.Info(ctx, "usecase.issue_message.AddMessage", "id", id)
	return id, nil
}

func (s *issueMessageService) GetByIssueID(ctx context.Context, issueID int64) ([]domain.IssueMessage, error) {
	logger.Info(ctx, "usecase.issue_message.GetByIssueID", "issue_id", issueID)

	msgs, err := s.msgRepo.GetByIssueID(ctx, issueID)
	if err != nil {
		logger.Error(ctx, "usecase.issue_message.GetByIssueID", "error", err)
		return nil, err
	}

	return msgs, nil
}

func (s *issueService) validateIssue(issue *domain.Issue) error {
	if !s.isValidCategory(issue.Category) {
		return domain.ErrInvalidInput
	}

	if issue.Status != "" && !s.isValidStatus(issue.Status) {
		return domain.ErrInvalidInput
	}

	if utf8.RuneCountInString(issue.Content) > 5000 {
		return domain.ErrInvalidInput
	}

	return nil
}

func (s *issueService) isValidCategory(category domain.IssueCategory) bool {
	switch category {
	case domain.CategoryBug, domain.CategoryIdea, domain.CategoryProblem, domain.CategoryFeedback:
		return true
	}
	return false
}

func (s *issueService) isValidStatus(status domain.IssueStatus) bool {
	switch status {
	case domain.IssueStatusOpen, domain.IssueStatusClosed, domain.IssueStatusInWork:
		return true
	}
	return false
}
