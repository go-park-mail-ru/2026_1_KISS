package usecase

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/sanitize"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

// Publisher is the interface for publishing notebook events to connected clients.
type Publisher interface {
	Publish(notebookID int64, event *pb.NotebookEvent)
}

type noopPublisher struct{}

func (noopPublisher) Publish(_ int64, _ *pb.NotebookEvent) {}

type NotebookService interface {
	Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error)
	GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error)
	Delete(ctx context.Context, userID, notebookID int64) error
	Update(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error)
	AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error)
	UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error)
	DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error
	ListAll(ctx context.Context, limit, offset int, search string) ([]domain.Notebook, error)
	CountAll(ctx context.Context, search string) (int, error)
	AdminDelete(ctx context.Context, notebookID int64) error

	GrantPermission(ctx context.Context, requesterID, notebookID, targetUserID int64, level string) error
	RevokePermission(ctx context.Context, requesterID, notebookID, targetUserID int64) error
	ListPermissions(ctx context.Context, requesterID, notebookID int64) ([]domain.FilePermission, error)
	ListSharedWithUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, int, error)
	SetAllPrivateByOwner(ctx context.Context, ownerID int64) error
}

type notebookService struct {
	notebookRepo repository.NotebookRepository
	blockRepo    repository.BlockRepository
	permRepo     repository.PermissionRepository
	publisher    Publisher
}

func New(nr repository.NotebookRepository, br repository.BlockRepository, pr repository.PermissionRepository, pubs ...Publisher) NotebookService {
	var pub Publisher = noopPublisher{}
	if len(pubs) > 0 && pubs[0] != nil {
		pub = pubs[0]
	}
	return &notebookService{
		notebookRepo: nr,
		blockRepo:    br,
		permRepo:     pr,
		publisher:    pub,
	}
}

func (s *notebookService) Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error) {
	logger.Info(ctx, "usecase.notebook.Create", "user_id", userID, "title", title)

	title = sanitize.EscapeHTML(title)
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
		_, err := s.permRepo.GetPermission(ctx, notebookID, userID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				logger.Error(ctx, "usecase.notebook.GetByID", "error", domain.ErrForbidden)
				return nil, domain.ErrForbidden
			}
			logger.Error(ctx, "usecase.notebook.GetByID", "error", err)
			return nil, err
		}
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

	title = sanitize.EscapeHTML(title)
	if title == "" {
		logger.Error(ctx, "usecase.notebook.Update", "error", domain.ErrInvalidInput)
		return nil, domain.ErrInvalidInput
	}
	if utf8.RuneCountInString(title) > 255 {
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
	s.publisher.Publish(nb.ID, &pb.NotebookEvent{
		Type:       pb.NotebookEvent_NOTEBOOK_UPDATED,
		NotebookId: nb.ID,
		ActorId:    userID,
		Timestamp:  time.Now().UnixMilli(),
	})
	return nb, nil
}

func (s *notebookService) AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error) {
	logger.Info(ctx, "usecase.notebook.AddBlock", "user_id", userID, "notebook_id", notebookID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.AddBlock", "error", err)
		return nil, err
	}
	if err := s.requireEditorAccess(ctx, nb, userID); err != nil {
		logger.Error(ctx, "usecase.notebook.AddBlock", "error", err)
		return nil, err
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
	s.publisher.Publish(notebookID, &pb.NotebookEvent{
		Type:       pb.NotebookEvent_BLOCK_ADDED,
		NotebookId: notebookID,
		ActorId:    userID,
		Timestamp:  time.Now().UnixMilli(),
		Payload:    &pb.NotebookEvent_Block{Block: blockToProto(block)},
	})
	return block, nil
}

func (s *notebookService) UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
	logger.Info(ctx, "usecase.notebook.UpdateBlock", "user_id", userID, "notebook_id", notebookID, "block_id", blockID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.UpdateBlock", "error", err)
		return nil, err
	}
	if err := s.requireEditorAccess(ctx, nb, userID); err != nil {
		logger.Error(ctx, "usecase.notebook.UpdateBlock", "error", err)
		return nil, err
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
	s.publisher.Publish(notebookID, &pb.NotebookEvent{
		Type:       pb.NotebookEvent_BLOCK_UPDATED,
		NotebookId: notebookID,
		ActorId:    userID,
		Timestamp:  time.Now().UnixMilli(),
		Payload:    &pb.NotebookEvent_Block{Block: blockToProto(block)},
	})
	return block, nil
}

func (s *notebookService) DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error {
	logger.Info(ctx, "usecase.notebook.DeleteBlock", "user_id", userID, "notebook_id", notebookID, "block_id", blockID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.DeleteBlock", "error", err)
		return err
	}
	if err := s.requireEditorAccess(ctx, nb, userID); err != nil {
		logger.Error(ctx, "usecase.notebook.DeleteBlock", "error", err)
		return err
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
	s.publisher.Publish(notebookID, &pb.NotebookEvent{
		Type:       pb.NotebookEvent_BLOCK_DELETED,
		NotebookId: notebookID,
		ActorId:    userID,
		Timestamp:  time.Now().UnixMilli(),
		Payload:    &pb.NotebookEvent_DeletedBlockId{DeletedBlockId: blockID},
	})
	return nil
}

func (s *notebookService) ListAll(ctx context.Context, limit, offset int, search string) ([]domain.Notebook, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.notebookRepo.ListAll(ctx, limit, offset, search)
}

func (s *notebookService) CountAll(ctx context.Context, search string) (int, error) {
	return s.notebookRepo.CountAll(ctx, search)
}

func (s *notebookService) AdminDelete(ctx context.Context, notebookID int64) error {
	return s.notebookRepo.Delete(ctx, notebookID)
}

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

func (s *notebookService) GrantPermission(ctx context.Context, requesterID, notebookID, targetUserID int64, level string) error {
	logger.Info(ctx, "usecase.notebook.GrantPermission", "requester_id", requesterID, "notebook_id", notebookID, "target_user_id", targetUserID)

	if level != domain.PermissionReadOnly && level != domain.PermissionEditor {
		return domain.ErrInvalidInput
	}
	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.GrantPermission", "error", err)
		return err
	}
	if nb.OwnerID != requesterID {
		logger.Error(ctx, "usecase.notebook.GrantPermission", "error", domain.ErrForbidden)
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

func (s *notebookService) RevokePermission(ctx context.Context, requesterID, notebookID, targetUserID int64) error {
	logger.Info(ctx, "usecase.notebook.RevokePermission", "requester_id", requesterID, "notebook_id", notebookID, "target_user_id", targetUserID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.RevokePermission", "error", err)
		return err
	}
	if nb.OwnerID != requesterID {
		logger.Error(ctx, "usecase.notebook.RevokePermission", "error", domain.ErrForbidden)
		return domain.ErrForbidden
	}
	return s.permRepo.Delete(ctx, notebookID, targetUserID)
}

func (s *notebookService) ListPermissions(ctx context.Context, requesterID, notebookID int64) ([]domain.FilePermission, error) {
	logger.Info(ctx, "usecase.notebook.ListPermissions", "requester_id", requesterID, "notebook_id", notebookID)

	nb, err := s.notebookRepo.GetByID(ctx, notebookID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.ListPermissions", "error", err)
		return nil, err
	}
	if nb.OwnerID != requesterID {
		logger.Error(ctx, "usecase.notebook.ListPermissions", "error", domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}
	return s.permRepo.GetByNotebookID(ctx, notebookID)
}

func (s *notebookService) ListSharedWithUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, int, error) {
	logger.Info(ctx, "usecase.notebook.ListSharedWithUser", "user_id", userID, "limit", limit, "offset", offset)

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	notebooks, err := s.notebookRepo.GetSharedWithUser(ctx, userID, limit, offset)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.ListSharedWithUser", "error", err)
		return nil, 0, err
	}
	total, err := s.notebookRepo.CountSharedWithUser(ctx, userID)
	if err != nil {
		logger.Error(ctx, "usecase.notebook.ListSharedWithUser", "error", err)
		return nil, 0, err
	}
	logger.Info(ctx, "usecase.notebook.ListSharedWithUser", "total", total)
	return notebooks, total, nil
}

func blockToProto(b *domain.Block) *pb.BlockInfo {
	return &pb.BlockInfo{
		Id:         b.ID,
		NotebookId: b.NotebookID,
		Type:       b.Type,
		Language:   b.Language,
		Content:    b.Content,
		Position:   int32(b.Position), //nolint:gosec
		CreatedAt:  b.CreatedAt.Unix(),
		UpdatedAt:  b.UpdatedAt.Unix(),
	}
}

func (s *notebookService) SetAllPrivateByOwner(ctx context.Context, ownerID int64) error {
	return s.notebookRepo.SetAllPrivateByOwner(ctx, ownerID)
}
