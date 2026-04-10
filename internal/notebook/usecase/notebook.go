package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
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
	if title == "" {
		title = "Untitled"
	}
	nb := &domain.Notebook{OwnerID: userID, Title: title}
	id, err := s.notebookRepo.Create(ctx, nb)
	if err != nil {
		return nil, err
	}
	nb.ID = id
	return nb, nil
}

func (s *notebookService) GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID && !nb.IsPublic {
		return nil, domain.ErrForbidden
	}
	blocks, err := s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	nb.Blocks = blocks
	return nb, nil
}

func (s *notebookService) ListByUser(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	notebooks, err := s.notebookRepo.GetByOwnerID(ctx, userID, limit, offset, search)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.notebookRepo.CountByOwnerID(ctx, userID, search)
	if err != nil {
		return nil, 0, err
	}
	return notebooks, total, nil
}

func (s *notebookService) Delete(ctx context.Context, userID, notebookID int64) error {
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return err
	}
	if nb.OwnerID != userID {
		return domain.ErrForbidden
	}
	return s.notebookRepo.Delete(ctx, notebookID)
}

func (s *notebookService) Update(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error) {
	if title == "" {
		return nil, domain.ErrInvalidInput
	}
	if len(title) > 255 {
		return nil, domain.ErrInvalidInput
	}
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	nb.Title = title
	nb.IsPublic = isPublic
	if err := s.notebookRepo.Update(ctx, nb); err != nil {
		return nil, err
	}
	return nb, nil
}

func (s *notebookService) AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error) {
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	blocks, err := s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	block.NotebookID = notebookID
	block.Position = len(blocks)
	if block.Type == "text" && block.Language == "" {
		block.Language = "markdown"
	}
	id, err := s.blockRepo.Create(ctx, block)
	if err != nil {
		return nil, err
	}
	block.ID = id
	return block, nil
}

func (s *notebookService) UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	block, err := s.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		return nil, err
	}
	if block.NotebookID != notebookID {
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
		return nil, err
	}
	return block, nil
}

func (s *notebookService) DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error {
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return err
	}
	if nb.OwnerID != userID {
		return domain.ErrForbidden
	}
	block, err := s.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		return err
	}
	if block.NotebookID != notebookID {
		return domain.ErrNotFound
	}
	return s.blockRepo.Delete(ctx, blockID)
}
