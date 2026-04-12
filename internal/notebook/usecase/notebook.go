package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type NotebookService interface {
	Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error)
	GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error)
	Delete(ctx context.Context, userID, notebookID int64) error
	Update(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error)
	AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error)
	UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error)
	DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error
}

type notebookService struct {
	notebookRepo repository.NotebookRepository
	blockRepo    repository.BlockRepository
}

func New(nr repository.NotebookRepository, br repository.BlockRepository) NotebookService {
	return &notebookService{
		notebookRepo: nr,
		blockRepo:    br,
	}
}

func (s *notebookService) Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error) {
	logger.Info(ctx, "usecase.notebook.Create", "user_id", userID, "title", title)

	if title == "" {
		title = "Untitled"
	}
	nb := &domain.Notebook{OwnerID: userID, Title: title}
	id, err := s.notebookRepo.Create(ctx, nb)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.Create", "error", err)
		return nil, err
	}
	nb.ID = id
	logger.Info(ctx, "usecase.notebook.Create", "notebook_id", nb.ID)
	return nb, nil
}

func (s *notebookService) GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
	logger.Info(ctx, "usecase.notebook.GetByID", "user_id", userID, "notebook_id", notebookID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.GetByID", "error", err)
		return nil, err
	}
	if nb.OwnerID != userID && !nb.IsPublic {
		logger.Error(ctx, "usecase.notebook.GetByID", "error", domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}
	blocks, err := s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.GetByID", "error", err)
		return nil, err
	}
	nb.Blocks = blocks
	logger.Info(ctx, "usecase.notebook.GetByID", "notebook_id", nb.ID)
	return nb, nil
}

func (s *notebookService) ListByUser(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
	logger.Info(ctx, "usecase.notebook.ListByUser", "user_id", userID, "limit", limit, "offset", offset, "search", search)

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	notebooks, err := s.notebookRepo.GetByOwnerID(ctx, userID, limit, offset, search)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.ListByUser", "error", err)
		return nil, 0, err
	}
	total, err := s.notebookRepo.CountByOwnerID(ctx, userID, search)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.ListByUser", "error", err)
		return nil, 0, err
	}
	logger.Info(ctx, "usecase.notebook.ListByUser", "total", total)
	return notebooks, total, nil
}

func (s *notebookService) Delete(ctx context.Context, userID, notebookID int64) error {
	logger.Info(ctx, "usecase.notebook.Delete", "user_id", userID, "notebook_id", notebookID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.Delete", "error", err)
		return err
	}
	if nb.OwnerID != userID {
		logger.Error(ctx, "usecase.notebook.Delete", "error", domain.ErrForbidden)
		return domain.ErrForbidden
	}
	if err := s.notebookRepo.Delete(ctx, notebookID); err != nil {
		logger.Error(ctx, "usecase.notebook.Delete", "error", err)
		return err
	}
	logger.Info(ctx, "usecase.notebook.Delete", "notebook_id", notebookID, "status", "ok")
	return nil
}

func (s *notebookService) Update(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error) {
	logger.Info(ctx, "usecase.notebook.Update", "user_id", userID, "notebook_id", notebookID)

	if title == "" {
		logger.Error(ctx, "usecase.notebook.Update", "error", domain.ErrInvalidInput)
		return nil, domain.ErrInvalidInput
	}
	if len(title) > 255 {
		logger.Error(ctx, "usecase.notebook.Update", "error", domain.ErrInvalidInput)
		return nil, domain.ErrInvalidInput
	}
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.Update", "error", err)
		return nil, err
	}
	if nb.OwnerID != userID {
		logger.Error(ctx, "usecase.notebook.Update", "error", domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}
	nb.Title = title
	nb.IsPublic = isPublic
	if err := s.notebookRepo.Update(ctx, nb); err != nil {
		logger.Error(ctx, "usecase.notebook.Update", "error", err)
		return nil, err
	}
	logger.Info(ctx, "usecase.notebook.Update", "notebook_id", nb.ID)
	return nb, nil
}

func (s *notebookService) AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error) {
	logger.Info(ctx, "usecase.notebook.AddBlock", "user_id", userID, "notebook_id", notebookID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.AddBlock", "error", err)
		return nil, err
	}
	if nb.OwnerID != userID {
		logger.Error(ctx, "usecase.notebook.AddBlock", "error", domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}
	blocks, err := s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.AddBlock", "error", err)
		return nil, err
	}
	block.NotebookID = notebookID
	block.Position = len(blocks)
	if block.Type == "text" && block.Language == "" {
		block.Language = "markdown"
	}
	id, err := s.blockRepo.Create(ctx, block)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.AddBlock", "error", err)
		return nil, err
	}
	block.ID = id
	logger.Info(ctx, "usecase.notebook.AddBlock", "block_id", block.ID)
	return block, nil
}

func (s *notebookService) UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
	logger.Info(ctx, "usecase.notebook.UpdateBlock", "user_id", userID, "notebook_id", notebookID, "block_id", blockID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.UpdateBlock", "error", err)
		return nil, err
	}
	if nb.OwnerID != userID {
		logger.Error(ctx, "usecase.notebook.UpdateBlock", "error", domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}
	block, err := s.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.UpdateBlock", "error", err)
		return nil, err
	}
	if block.NotebookID != notebookID {
		logger.Error(ctx, "usecase.notebook.UpdateBlock", "error", domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}
	block.Content = content
	if cellType != "" {
		block.Type = cellType
	}
	if language != "" {
		block.Language = language
	}
	if err := s.blockRepo.Update(ctx, block); err != nil {
		logger.Error(ctx, "usecase.notebook.UpdateBlock", "error", err)
		return nil, err
	}
	logger.Info(ctx, "usecase.notebook.UpdateBlock", "block_id", block.ID)
	return block, nil
}

func (s *notebookService) DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error {
	logger.Info(ctx, "usecase.notebook.DeleteBlock", "user_id", userID, "notebook_id", notebookID, "block_id", blockID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.DeleteBlock", "error", err)
		return err
	}
	if nb.OwnerID != userID {
		logger.Error(ctx, "usecase.notebook.DeleteBlock", "error", domain.ErrForbidden)
		return domain.ErrForbidden
	}
	block, err := s.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.DeleteBlock", "error", err)
		return err
	}
	if block.NotebookID != notebookID {
		logger.Error(ctx, "usecase.notebook.DeleteBlock", "error", domain.ErrNotFound)
		return domain.ErrNotFound
	}
	if err := s.blockRepo.Delete(ctx, blockID); err != nil {
		logger.Error(ctx, "usecase.notebook.DeleteBlock", "error", err)
		return err
	}
	logger.Info(ctx, "usecase.notebook.DeleteBlock", "block_id", blockID, "status", "ok")
	return nil
}
