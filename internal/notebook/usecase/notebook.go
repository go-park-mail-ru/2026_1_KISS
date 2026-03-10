package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
)

type NotebookUsecase struct {
	notebookRepo repository.NotebookRepository
	blockRepo    repository.BlockRepository
}

func New(nr repository.NotebookRepository, br repository.BlockRepository) *NotebookUsecase {
	return &NotebookUsecase{
		notebookRepo: nr,
		blockRepo:    br,
	}
}

func (uc *NotebookUsecase) Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error) {
	if title == "" {
		title = "Untitled"
	}
	nb := &domain.Notebook{OwnerID: userID, Title: title}
	id, err := uc.notebookRepo.Create(ctx, nb)
	if err != nil {
		return nil, err
	}
	nb.ID = id
	return nb, nil
}

func (uc *NotebookUsecase) GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
	nb, err := uc.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID && !nb.IsPublic {
		return nil, domain.ErrForbidden
	}
	blocks, err := uc.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	nb.Blocks = blocks
	return nb, nil
}

func (uc *NotebookUsecase) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return uc.notebookRepo.GetByOwnerID(ctx, userID, limit, offset)
}

func (uc *NotebookUsecase) Delete(ctx context.Context, userID, notebookID int64) error {
	nb, err := uc.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return err
	}
	if nb.OwnerID != userID {
		return domain.ErrForbidden
	}
	return uc.notebookRepo.Delete(ctx, notebookID)
}

func (uc *NotebookUsecase) Update(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error) {
	if title == "" {
		return nil, domain.ErrInvalidInput
	}
	if len(title) > 255 {
		return nil, domain.ErrInvalidInput
	}
	nb, err := uc.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	nb.Title = title
	nb.IsPublic = isPublic
	if err := uc.notebookRepo.Update(ctx, nb); err != nil {
		return nil, err
	}
	return nb, nil
}

func (uc *NotebookUsecase) AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error) {
	nb, err := uc.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	blocks, err := uc.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	block.NotebookID = notebookID
	block.Position = len(blocks)
	id, err := uc.blockRepo.Create(ctx, block)
	if err != nil {
		return nil, err
	}
	block.ID = id
	return block, nil
}

func (uc *NotebookUsecase) UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
	nb, err := uc.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	block, err := uc.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		return nil, err
	}
	if block.NotebookID != notebookID {
		return nil, domain.ErrNotFound
	}
	block.Content = content
	block.Type = cellType
	block.Language = language
	if err := uc.blockRepo.Update(ctx, block); err != nil {
		return nil, err
	}
	return block, nil
}

func (uc *NotebookUsecase) DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error {
	nb, err := uc.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return err
	}
	if nb.OwnerID != userID {
		return domain.ErrForbidden
	}
	block, err := uc.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		return err
	}
	if block.NotebookID != notebookID {
		return domain.ErrNotFound
	}
	return uc.blockRepo.Delete(ctx, blockID)
}
