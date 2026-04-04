package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
)

type NotebookService interface {
	Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error)
	GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, int, error)
	Delete(ctx context.Context, userID, notebookID int64) error
	Update(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error)
	AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error)
	UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error)
	DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error

	GrantPermission(ctx context.Context, requesterID, notebookID, targetUserID int64, level string) error
	RevokePermission(ctx context.Context, requesterID, notebookID, targetUserID int64) error
	ListPermissions(ctx context.Context, requesterID, notebookID int64) ([]domain.FilePermission, error)
	ListSharedWithUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, int, error)
}

type notebookService struct {
	notebookRepo repository.NotebookRepository
	blockRepo    repository.BlockRepository
	permRepo     repository.PermissionRepository
}

func New(nr repository.NotebookRepository, br repository.BlockRepository, pr repository.PermissionRepository) NotebookService {
	return &notebookService{
		notebookRepo: nr,
		blockRepo:    br,
		permRepo:     pr,
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
		_, err := s.permRepo.GetPermission(ctx, notebookID, userID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ErrForbidden
			}
			return nil, err
		}
	}
	blocks, err := s.blockRepo.GetByNotebookID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	nb.Blocks = blocks
	return nb, nil
}

func (s *notebookService) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	notebooks, err := s.notebookRepo.GetByOwnerID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.notebookRepo.CountByOwnerID(ctx, userID)
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
	if err := s.requireEditorAccess(ctx, nb, userID); err != nil {
		return nil, err
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
	if err := s.requireEditorAccess(ctx, nb, userID); err != nil {
		return nil, err
	}
	block, err := s.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		return nil, err
	}
	if block.NotebookID != notebookID {
		return nil, domain.ErrNotFound
	}
	block.Content = content
	block.Type = cellType
	block.Language = language
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
	if err := s.requireEditorAccess(ctx, nb, userID); err != nil {
		return err
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

// requireEditorAccess проверяет, что пользователь является владельцем
// или имеет уровень доступа "editor".
func (s *notebookService) requireEditorAccess(ctx context.Context, nb *domain.Notebook, userID int64) error {
	if nb.OwnerID == userID {
		return nil
	}
	perm, err := s.permRepo.GetPermission(ctx, nb.ID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	if perm.PermissionLevel != domain.PermissionEditor {
		return domain.ErrForbidden
	}
	return nil
}

// ListSharedWithUser возвращает ноутбуки, к которым у пользователя есть явное разрешение (не его собственные).
func (s *notebookService) ListSharedWithUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	notebooks, err := s.notebookRepo.GetSharedWithUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.notebookRepo.CountSharedWithUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return notebooks, total, nil
}

// GrantPermission выдаёт или обновляет разрешение на notebook для targetUserID.
// Только владелец может управлять разрешениями.
func (s *notebookService) GrantPermission(ctx context.Context, requesterID, notebookID, targetUserID int64, level string) error {
	if level != domain.PermissionReadOnly && level != domain.PermissionEditor {
		return domain.ErrInvalidInput
	}
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return err
	}
	if nb.OwnerID != requesterID {
		return domain.ErrForbidden
	}
	if targetUserID == nb.OwnerID {
		return domain.ErrInvalidInput
	}
	return s.permRepo.Upsert(ctx, &domain.FilePermission{
		NotebookID:      notebookID,
		UserID:          targetUserID,
		PermissionLevel: level,
	})
}

// RevokePermission удаляет разрешение targetUserID на notebook.
// Только владелец может управлять разрешениями.
func (s *notebookService) RevokePermission(ctx context.Context, requesterID, notebookID, targetUserID int64) error {
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return err
	}
	if nb.OwnerID != requesterID {
		return domain.ErrForbidden
	}
	return s.permRepo.Delete(ctx, notebookID, targetUserID)
}

// ListPermissions возвращает все разрешения для notebook.
// Только владелец может просматривать список.
func (s *notebookService) ListPermissions(ctx context.Context, requesterID, notebookID int64) ([]domain.FilePermission, error) {
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		return nil, err
	}
	if nb.OwnerID != requesterID {
		return nil, domain.ErrForbidden
	}
	return s.permRepo.GetByNotebookID(ctx, notebookID)
}
