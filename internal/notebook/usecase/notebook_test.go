package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
)

type mockNotebookRepo struct {
	createFn       func(ctx context.Context, nb *domain.Notebook) (int64, error)
	getByIDFn      func(ctx context.Context, id int64) (*domain.Notebook, error)
	getByOwnerIDFn func(ctx context.Context, ownerID int64, limit, offset int) ([]domain.Notebook, error)
	deleteFn       func(ctx context.Context, id int64) error
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

func (m *mockNotebookRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type mockBlockRepo struct {
	createFn          func(ctx context.Context, b *domain.Block) (int64, error)
	getByNotebookIDFn func(ctx context.Context, notebookID int64) ([]domain.Block, error)
}

func (m *mockBlockRepo) Create(ctx context.Context, b *domain.Block) (int64, error) {
	if m.createFn != nil {
		return m.createFn(ctx, b)
	}
	return 0, nil
}

func (m *mockBlockRepo) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.Block, error) {
	if m.getByNotebookIDFn != nil {
		return m.getByNotebookIDFn(ctx, notebookID)
	}
	return []domain.Block{}, nil
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
	}
	uc := usecase.New(nbRepo, &mockBlockRepo{})
	_, err := uc.ListByUser(context.Background(), 1, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo not called")
	}
}
