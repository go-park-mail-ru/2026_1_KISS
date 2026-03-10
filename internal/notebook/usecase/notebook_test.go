package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
)

type mockNotebookRepo struct {
	createFn        func(ctx context.Context, nb *domain.Notebook) (int64, error)
	getByIDFn       func(ctx context.Context, id int64) (*domain.Notebook, error)
	getByOwnerIDFn  func(ctx context.Context, ownerID int64, limit, offset int) ([]domain.Notebook, error)
	updateFn        func(ctx context.Context, nb *domain.Notebook) error
	deleteFn        func(ctx context.Context, id int64) error
	countByOwnerIDFn func(ctx context.Context, ownerID int64) (int, error)
}

func (m *mockNotebookRepo) Create(ctx context.Context, nb *domain.Notebook) (int64, error) {
	if m.createFn != nil {
		return m.createFn(ctx, nb)
	}
	return 0, nil
}

func (m *mockNotebookRepo) GetByID(ctx context.Context, id int64) (*domain.Notebook, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockNotebookRepo) GetByOwnerID(ctx context.Context, ownerID int64, limit, offset int) ([]domain.Notebook, error) {
	if m.getByOwnerIDFn != nil {
		return m.getByOwnerIDFn(ctx, ownerID, limit, offset)
	}
	return []domain.Notebook{}, nil
}

func (m *mockNotebookRepo) Update(ctx context.Context, nb *domain.Notebook) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, nb)
	}
	return nil
}

func (m *mockNotebookRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockNotebookRepo) CountByOwnerID(ctx context.Context, ownerID int64) (int, error) {
	if m.countByOwnerIDFn != nil {
		return m.countByOwnerIDFn(ctx, ownerID)
	}
	return 0, nil
}

type mockBlockRepo struct {
	createFn          func(ctx context.Context, b *domain.Block) (int64, error)
	getByIDFn         func(ctx context.Context, blockID int64) (*domain.Block, error)
	getByNotebookIDFn func(ctx context.Context, notebookID int64) ([]domain.Block, error)
	updateFn          func(ctx context.Context, b *domain.Block) error
	deleteFn          func(ctx context.Context, blockID int64) error
}

func (m *mockBlockRepo) Create(ctx context.Context, b *domain.Block) (int64, error) {
	if m.createFn != nil {
		return m.createFn(ctx, b)
	}
	return 0, nil
}

func (m *mockBlockRepo) GetByID(ctx context.Context, blockID int64) (*domain.Block, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, blockID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockBlockRepo) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.Block, error) {
	if m.getByNotebookIDFn != nil {
		return m.getByNotebookIDFn(ctx, notebookID)
	}
	return []domain.Block{}, nil
}

func (m *mockBlockRepo) Update(ctx context.Context, b *domain.Block) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, b)
	}
	return nil
}

func (m *mockBlockRepo) Delete(ctx context.Context, blockID int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, blockID)
	}
	return nil
}

func TestCreate_DefaultTitle(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		createFn: func(ctx context.Context, nb *domain.Notebook) (int64, error) {
			if nb.Title != "Untitled" {
				t.Errorf("want Untitled, got %s", nb.Title)
			}
			return 1, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	nb, err := uc.Create(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nb.ID != 1 {
		t.Errorf("want ID=1, got %d", nb.ID)
	}
}

func TestGetByID_Forbidden(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 2, IsPublic: false}, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.GetByID(context.Background(), 1, 42)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestGetByID_PublicAccess(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 2, IsPublic: true}, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	nb, err := uc.GetByID(context.Background(), 1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nb == nil {
		t.Error("expected notebook")
	}
}

func TestDelete_Forbidden(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 2}, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	err := uc.Delete(context.Background(), 1, 42)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestAddBlock_CorrectPosition(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByNotebookIDFn: func(ctx context.Context, notebookID int64) ([]domain.Block, error) {
			return []domain.Block{{ID: 1}, {ID: 2}}, nil
		},
		createFn: func(ctx context.Context, b *domain.Block) (int64, error) {
			if b.Position != 2 {
				t.Errorf("want position 2, got %d", b.Position)
			}
			return 3, nil
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	block, err := uc.AddBlock(context.Background(), 1, 1, &domain.Block{Type: "code"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if block.ID != 3 {
		t.Errorf("want ID=3, got %d", block.ID)
	}
}

func TestListByUser_NormalizesLimit(t *testing.T) {
	called := false
	nbRepo := &mockNotebookRepo{
		getByOwnerIDFn: func(ctx context.Context, ownerID int64, limit, offset int) ([]domain.Notebook, error) {
			called = true
			if limit != 20 {
				t.Errorf("want limit=20, got %d", limit)
			}
			return []domain.Notebook{}, nil
		},
		countByOwnerIDFn: func(ctx context.Context, ownerID int64) (int, error) {
			return 0, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, total, err := uc.ListByUser(context.Background(), 1, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo not called")
	}
	if total != 0 {
		t.Errorf("want total=0, got %d", total)
	}
}

func TestCreate_RepoError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		createFn: func(ctx context.Context, nb *domain.Notebook) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.Create(context.Background(), 1, "test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetByID_RepoError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return nil, errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.GetByID(context.Background(), 1, 42)
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetByID_BlocksError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByNotebookIDFn: func(ctx context.Context, notebookID int64) ([]domain.Block, error) {
			return nil, errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	_, err := uc.GetByID(context.Background(), 1, 1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestListByUser_NegativeOffset(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByOwnerIDFn: func(ctx context.Context, ownerID int64, limit, offset int) ([]domain.Notebook, error) {
			if offset != 0 {
				t.Errorf("want offset=0, got %d", offset)
			}
			return []domain.Notebook{}, nil
		},
		countByOwnerIDFn: func(ctx context.Context, ownerID int64) (int, error) {
			return 0, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, _, err := uc.ListByUser(context.Background(), 1, 10, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			return nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	err := uc.Delete(context.Background(), 1, 42)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelete_GetByIDError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return nil, errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	err := uc.Delete(context.Background(), 1, 42)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDelete_RepoError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			return errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	err := uc.Delete(context.Background(), 1, 42)
	if err == nil {
		t.Error("expected error")
	}
}

func TestAddBlock_NotebookNotFound(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.AddBlock(context.Background(), 1, 42, &domain.Block{Type: "code"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestAddBlock_Forbidden(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 2}, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.AddBlock(context.Background(), 1, 42, &domain.Block{Type: "code"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestAddBlock_GetBlocksError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByNotebookIDFn: func(ctx context.Context, notebookID int64) ([]domain.Block, error) {
			return nil, errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	_, err := uc.AddBlock(context.Background(), 1, 1, &domain.Block{Type: "code"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestAddBlock_CreateError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		createFn: func(ctx context.Context, b *domain.Block) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	_, err := uc.AddBlock(context.Background(), 1, 1, &domain.Block{Type: "code"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestUpdate_Success(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1, Title: "Old"}, nil
		},
		updateFn: func(ctx context.Context, nb *domain.Notebook) error {
			if nb.Title != "New Title" {
				t.Errorf("want title New Title, got %s", nb.Title)
			}
			if !nb.IsPublic {
				t.Error("want IsPublic=true")
			}
			return nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	nb, err := uc.Update(context.Background(), 1, 42, "New Title", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nb.Title != "New Title" {
		t.Errorf("want New Title, got %s", nb.Title)
	}
}

func TestUpdate_EmptyTitle(t *testing.T) {
	uc := usecase.New(&mockNotebookRepo{}, &mockBlockRepo{})
	_, err := uc.Update(context.Background(), 1, 42, "", false)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_TooLongTitle(t *testing.T) {
	longTitle := string(make([]byte, 256))
	uc := usecase.New(&mockNotebookRepo{}, &mockBlockRepo{})
	_, err := uc.Update(context.Background(), 1, 42, longTitle, false)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_Forbidden(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 2}, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.Update(context.Background(), 1, 42, "Title", false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.Update(context.Background(), 1, 42, "Title", false)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_Success(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return &domain.Block{ID: blockID, NotebookID: 10, Type: "code"}, nil
		},
		updateFn: func(ctx context.Context, b *domain.Block) error {
			if b.Content != "new content" {
				t.Errorf("want new content, got %s", b.Content)
			}
			return nil
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	block, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "new content", "markdown", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if block.Type != "markdown" {
		t.Errorf("want markdown, got %s", block.Type)
	}
}

func TestUpdateBlock_NotebookNotFound(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_Forbidden(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 2}, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestUpdateBlock_BlockNotFound(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_WrongNotebook(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return &domain.Block{ID: blockID, NotebookID: 999}, nil
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_RepoError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return &domain.Block{ID: blockID, NotebookID: 10}, nil
		},
		updateFn: func(ctx context.Context, b *domain.Block) error {
			return errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDeleteBlock_Success(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return &domain.Block{ID: blockID, NotebookID: 10}, nil
		},
		deleteFn: func(ctx context.Context, blockID int64) error {
			return nil
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteBlock_NotebookNotFound(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteBlock_Forbidden(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 2}, nil
		},
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestDeleteBlock_BlockNotFound(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteBlock_WrongNotebook(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return &domain.Block{ID: blockID, NotebookID: 999}, nil
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteBlock_RepoError(t *testing.T) {
	nbRepo := &mockNotebookRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: id, OwnerID: 1}, nil
		},
	}
	blockRepo := &mockBlockRepo{
		getByIDFn: func(ctx context.Context, blockID int64) (*domain.Block, error) {
			return &domain.Block{ID: blockID, NotebookID: 10}, nil
		},
		deleteFn: func(ctx context.Context, blockID int64) error {
			return errors.New("db error")
		},
	}
	uc := usecase.New(nbRepo, blockRepo)
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if err == nil {
		t.Error("expected error")
	}
}
