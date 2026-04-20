package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"go.uber.org/mock/gomock"
)

func TestCreate_DefaultTitle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, nb *domain.Notebook) (int64, error) {
			if nb.Title != "Untitled" {
				t.Errorf("want Untitled, got %s", nb.Title)
			}
			return 1, nil
		})

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	nb, err := uc.Create(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nb.ID != 1 {
		t.Errorf("want ID=1, got %d", nb.ID)
	}
}

func TestGetByID_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)
	permRepo := mocks.NewMockPermissionRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 2, IsPublic: false}, nil)
	permRepo.EXPECT().GetPermission(gomock.Any(), int64(42), int64(1)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, permRepo)
	_, err := uc.GetByID(context.Background(), 1, 42)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestGetByID_PublicAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 2, IsPublic: true}, nil)
	blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(42)).
		Return([]domain.Block{}, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	nb, err := uc.GetByID(context.Background(), 1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nb == nil {
		t.Error("expected notebook")
	}
}

func TestDelete_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 2}, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.Delete(context.Background(), 1, 42)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestAddBlock_CorrectPosition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).
		Return(&domain.Notebook{ID: 1, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).
		Return([]domain.Block{{ID: 1}, {ID: 2}}, nil)
	blockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, b *domain.Block) (int64, error) {
			if b.Position != 2 {
				t.Errorf("want position 2, got %d", b.Position)
			}
			return 3, nil
		})

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	block, err := uc.AddBlock(context.Background(), 1, 1, &domain.Block{Type: "code"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if block.ID != 3 {
		t.Errorf("want ID=3, got %d", block.ID)
	}
}

func TestAddBlock_TextDefaultsLanguageToPlain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).
		Return(&domain.Notebook{ID: 1, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).
		Return([]domain.Block{}, nil)
	blockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, b *domain.Block) (int64, error) {
			if b.Language != "markdown" {
				t.Errorf("want language markdown, got %q", b.Language)
			}
			return 1, nil
		})

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	block, err := uc.AddBlock(context.Background(), 1, 1, &domain.Block{Type: "text", Language: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if block.Language != "markdown" {
		t.Errorf("want language markdown, got %q", block.Language)
	}
}

func TestListByUser_NormalizesLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	called := false
	notebookRepo.EXPECT().GetByOwnerID(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), "").
		DoAndReturn(func(ctx context.Context, ownerID int64, limit, offset int, search string) ([]domain.Notebook, error) {
			called = true
			if limit != 20 {
				t.Errorf("want limit=20, got %d", limit)
			}
			return []domain.Notebook{}, nil
		})
	notebookRepo.EXPECT().CountByOwnerID(gomock.Any(), int64(1), "").
		Return(0, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, total, err := uc.ListByUser(context.Background(), 1, 0, 0, "")
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(int64(0), errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.Create(context.Background(), 1, "test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetByID_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(nil, errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.GetByID(context.Background(), 1, 42)
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetByID_BlocksError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).
		Return(&domain.Notebook{ID: 1, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).
		Return(nil, errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.GetByID(context.Background(), 1, 1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestListByUser_NegativeOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByOwnerID(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), "").
		DoAndReturn(func(ctx context.Context, ownerID int64, limit, offset int, search string) ([]domain.Notebook, error) {
			if offset != 0 {
				t.Errorf("want offset=0, got %d", offset)
			}
			return []domain.Notebook{}, nil
		})
	notebookRepo.EXPECT().CountByOwnerID(gomock.Any(), int64(1), "").
		Return(0, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, _, err := uc.ListByUser(context.Background(), 1, 10, -5, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListByUser_SearchPropagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	gotSearchRepo := ""
	gotSearchCount := ""
	notebookRepo.EXPECT().GetByOwnerID(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), "test").
		DoAndReturn(func(ctx context.Context, ownerID int64, limit, offset int, search string) ([]domain.Notebook, error) {
			gotSearchRepo = search
			return []domain.Notebook{{ID: 1, Title: "foo test bar"}}, nil
		})
	notebookRepo.EXPECT().CountByOwnerID(gomock.Any(), int64(1), "test").
		DoAndReturn(func(ctx context.Context, ownerID int64, search string) (int, error) {
			gotSearchCount = search
			return 1, nil
		})

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	notebooks, total, err := uc.ListByUser(context.Background(), 1, 10, 0, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSearchRepo != "test" {
		t.Errorf("search not propagated to GetByOwnerID: got %q", gotSearchRepo)
	}
	if gotSearchCount != "test" {
		t.Errorf("search not propagated to CountByOwnerID: got %q", gotSearchCount)
	}
	if len(notebooks) != 1 || total != 1 {
		t.Errorf("want 1 notebook and total=1, got %d notebooks, total=%d", len(notebooks), total)
	}
}

func TestListByUser_EmptySearchReturnsAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByOwnerID(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), "").
		DoAndReturn(func(ctx context.Context, ownerID int64, limit, offset int, search string) ([]domain.Notebook, error) {
			if search != "" {
				t.Errorf("want empty search, got %q", search)
			}
			return []domain.Notebook{{ID: 1}, {ID: 2}}, nil
		})
	notebookRepo.EXPECT().CountByOwnerID(gomock.Any(), int64(1), "").
		Return(2, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	notebooks, total, err := uc.ListByUser(context.Background(), 1, 10, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notebooks) != 2 || total != 2 {
		t.Errorf("want 2 notebooks, got %d, total=%d", len(notebooks), total)
	}
}

func TestDelete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 1}, nil)
	notebookRepo.EXPECT().Delete(gomock.Any(), int64(42)).
		Return(nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.Delete(context.Background(), 1, 42)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelete_GetByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(nil, errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.Delete(context.Background(), 1, 42)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDelete_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 1}, nil)
	notebookRepo.EXPECT().Delete(gomock.Any(), int64(42)).
		Return(errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.Delete(context.Background(), 1, 42)
	if err == nil {
		t.Error("expected error")
	}
}

func TestAddBlock_NotebookNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.AddBlock(context.Background(), 1, 42, &domain.Block{Type: "code"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestAddBlock_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)
	permRepo := mocks.NewMockPermissionRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 2}, nil)
	permRepo.EXPECT().GetPermission(gomock.Any(), int64(42), int64(1)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, permRepo)
	_, err := uc.AddBlock(context.Background(), 1, 42, &domain.Block{Type: "code"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestAddBlock_GetBlocksError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).
		Return(&domain.Notebook{ID: 1, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).
		Return(nil, errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.AddBlock(context.Background(), 1, 1, &domain.Block{Type: "code"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestAddBlock_CreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).
		Return(&domain.Notebook{ID: 1, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).
		Return([]domain.Block{}, nil)
	blockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(int64(0), errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.AddBlock(context.Background(), 1, 1, &domain.Block{Type: "code"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestUpdate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 1, Title: "Old"}, nil)
	notebookRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, nb *domain.Notebook) error {
			if nb.Title != "New Title" {
				t.Errorf("want title New Title, got %s", nb.Title)
			}
			if !nb.IsPublic {
				t.Error("want IsPublic=true")
			}
			return nil
		})

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	nb, err := uc.Update(context.Background(), 1, 42, "New Title", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nb.Title != "New Title" {
		t.Errorf("want New Title, got %s", nb.Title)
	}
}

func TestUpdate_EmptyTitle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.Update(context.Background(), 1, 42, "", false)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_TooLongTitle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	longTitle := string(make([]byte, 256))
	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.Update(context.Background(), 1, 42, longTitle, false)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(&domain.Notebook{ID: 42, OwnerID: 2}, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.Update(context.Background(), 1, 42, "Title", false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(42)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.Update(context.Background(), 1, 42, "Title", false)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&domain.Block{ID: 5, NotebookID: 10, Type: "code"}, nil)
	blockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, b *domain.Block) error {
			if b.Content != "new content" {
				t.Errorf("want new content, got %s", b.Content)
			}
			return nil
		})

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	block, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "new content", "markdown", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if block.Type != "markdown" {
		t.Errorf("want markdown, got %s", block.Type)
	}
}

func TestUpdateBlock_NotebookNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)
	permRepo := mocks.NewMockPermissionRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 2}, nil)
	permRepo.EXPECT().GetPermission(gomock.Any(), int64(10), int64(1)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, permRepo)
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestUpdateBlock_BlockNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_WrongNotebook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&domain.Block{ID: 5, NotebookID: 999}, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateBlock_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&domain.Block{ID: 5, NotebookID: 10}, nil)
	blockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
		Return(errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	_, err := uc.UpdateBlock(context.Background(), 1, 10, 5, "c", "code", "py")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDeleteBlock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&domain.Block{ID: 5, NotebookID: 10}, nil)
	blockRepo.EXPECT().Delete(gomock.Any(), int64(5)).
		Return(nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteBlock_NotebookNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteBlock_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)
	permRepo := mocks.NewMockPermissionRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 2}, nil)
	permRepo.EXPECT().GetPermission(gomock.Any(), int64(10), int64(1)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, permRepo)
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden, got %v", err)
	}
}

func TestDeleteBlock_BlockNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(nil, domain.ErrNotFound)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteBlock_WrongNotebook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&domain.Block{ID: 5, NotebookID: 999}, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteBlock_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().GetByID(gomock.Any(), int64(10)).
		Return(&domain.Notebook{ID: 10, OwnerID: 1}, nil)
	blockRepo.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&domain.Block{ID: 5, NotebookID: 10}, nil)
	blockRepo.EXPECT().Delete(gomock.Any(), int64(5)).
		Return(errors.New("db error"))

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.DeleteBlock(context.Background(), 1, 10, 5)
	if err == nil {
		t.Error("expected error")
	}
}

func TestListAll_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebooks := []domain.Notebook{{ID: 1, Title: "nb1"}, {ID: 2, Title: "nb2"}}
	notebookRepo.EXPECT().ListAll(gomock.Any(), 20, 0, "").
		Return(notebooks, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	result, err := uc.ListAll(context.Background(), 0, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("want 2 notebooks, got %d", len(result))
	}
}

func TestCountAll_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().CountAll(gomock.Any(), "").
		Return(42, nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	count, err := uc.CountAll(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("want 42, got %d", count)
	}
}

func TestAdminDelete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)

	notebookRepo.EXPECT().Delete(gomock.Any(), int64(42)).
		Return(nil)

	uc := usecase.New(notebookRepo, blockRepo, mocks.NewMockPermissionRepository(ctrl))
	err := uc.AdminDelete(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
