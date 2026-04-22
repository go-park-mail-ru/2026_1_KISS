package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	nbhttp "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/delivery/http"
)

type mockNotebookUsecase struct {
	createFn      func(ctx context.Context, userID int64, title string) (*domain.Notebook, error)
	getByIDFn     func(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error)
	listByUserFn  func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error)
	updateFn      func(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error)
	deleteFn      func(ctx context.Context, userID, notebookID int64) error
	addBlockFn    func(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error)
	updateBlockFn func(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error)
	deleteBlockFn func(ctx context.Context, userID, notebookID, blockID int64) error
}

func (m *mockNotebookUsecase) Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, title)
	}
	return nil, nil
}

func (m *mockNotebookUsecase) GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, userID, notebookID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockNotebookUsecase) ListByUser(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, limit, offset, search)
	}
	return []domain.Notebook{}, 0, nil
}

func (m *mockNotebookUsecase) Update(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, userID, notebookID, title, isPublic)
	}
	return nil, nil
}

func (m *mockNotebookUsecase) Delete(ctx context.Context, userID, notebookID int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID, notebookID)
	}
	return nil
}

func (m *mockNotebookUsecase) AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error) {
	if m.addBlockFn != nil {
		return m.addBlockFn(ctx, userID, notebookID, block)
	}
	return nil, nil
}

func (m *mockNotebookUsecase) UpdateBlock(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
	if m.updateBlockFn != nil {
		return m.updateBlockFn(ctx, userID, notebookID, blockID, content, cellType, language)
	}
	return nil, nil
}

func (m *mockNotebookUsecase) DeleteBlock(ctx context.Context, userID, notebookID, blockID int64) error {
	if m.deleteBlockFn != nil {
		return m.deleteBlockFn(ctx, userID, notebookID, blockID)
	}
	return nil
}

func reqWithUser(req *http.Request, user *domain.User) *http.Request {
	ctx := middleware.SetUserInContext(req.Context(), user)
	return req.WithContext(ctx)
}

var testUser = &domain.User{ID: 1, Username: "testuser", Email: "test@example.com"}

func TestList(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
			return []domain.Notebook{{ID: 1, Title: "Test"}}, 1, nil
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestList_SearchQueryParamPropagates(t *testing.T) {
	gotSearch := ""
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
			gotSearch = search
			return []domain.Notebook{}, 0, nil
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks?limit=7&offset=0&search=foo", nil)
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotSearch != "foo" {
		t.Errorf("search not propagated: got %q, want %q", gotSearch, "foo")
	}
}

func TestList_NoSearchQueryParamSendsEmpty(t *testing.T) {
	gotSearch := "not-set"
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
			gotSearch = search
			return []domain.Notebook{}, 0, nil
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks?limit=7&offset=0", nil)
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotSearch != "" {
		t.Errorf("want empty search, got %q", gotSearch)
	}
}

func TestCreate_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		createFn: func(ctx context.Context, userID int64, title string) (*domain.Notebook, error) {
			return &domain.Notebook{ID: 1, Title: title, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/notebooks", strings.NewReader(`{"title":"My Notebook"}`))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestGetByID_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		getByIDFn: func(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
			return &domain.Notebook{ID: notebookID, OwnerID: userID, Title: "Test"}, nil
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		getByIDFn: func(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
			return nil, domain.ErrNotFound
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks/99", nil)
	req.SetPathValue("id", "99")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestGetByID_Forbidden(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		getByIDFn: func(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
			return nil, domain.ErrForbidden
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestDelete_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		deleteFn: func(ctx context.Context, userID, notebookID int64) error { return nil },
	})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d", rec.Code)
	}
}

func TestAddBlock_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		addBlockFn: func(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error) {
			block.ID = 1
			return block, nil
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/blocks",
		strings.NewReader(`{"type":"code","language":"python","content":"print(1)"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.AddBlock(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestGetByID_InvalidID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("GET", "/api/v1/notebooks/abc", nil)
	req.SetPathValue("id", "abc")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestRegisterRoutes_Notebook(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	mux := http.NewServeMux()
	noopMw := middleware.Middleware(func(next http.Handler) http.Handler { return next })
	h.RegisterRoutes(mux, noopMw)
}

func TestGetByID_WithBlocks(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		getByIDFn: func(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error) {
			return &domain.Notebook{
				ID: notebookID, OwnerID: userID, Title: "Test",
				Blocks: []domain.Block{{ID: 1, Type: "code", Language: "python"}},
			}, nil
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestList_Error(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
			return nil, 0, errors.New("db error")
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestList_Conflict(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
			return nil, 0, domain.ErrConflict
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", rec.Code)
	}
}

func TestList_Unauthorized(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
			return nil, 0, domain.ErrUnauthorized
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestList_InvalidInput(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int, search string) ([]domain.Notebook, int, error) {
			return nil, 0, domain.ErrInvalidInput
		},
	})
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestCreate_InvalidBody(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("POST", "/api/v1/notebooks", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestCreate_Error(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		createFn: func(ctx context.Context, userID int64, title string) (*domain.Notebook, error) {
			return nil, errors.New("db error")
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/notebooks", strings.NewReader(`{"title":"Test"}`))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestDelete_InvalidID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/abc", nil)
	req.SetPathValue("id", "abc")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestDelete_Forbidden(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		deleteFn: func(ctx context.Context, userID, notebookID int64) error {
			return domain.ErrForbidden
		},
	})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		deleteFn: func(ctx context.Context, userID, notebookID int64) error {
			return domain.ErrNotFound
		},
	})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestAddBlock_InvalidID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/abc/blocks", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "abc")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.AddBlock(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAddBlock_InvalidBody(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/blocks", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.AddBlock(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAddBlock_Error(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		addBlockFn: func(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error) {
			return nil, domain.ErrForbidden
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/blocks",
		strings.NewReader(`{"type":"code","language":"python","content":"print(1)"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.AddBlock(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestUpdate_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		updateFn: func(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error) {
			return &domain.Notebook{ID: notebookID, OwnerID: userID, Title: title, IsPublic: isPublic}, nil
		},
	})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1", strings.NewReader(`{"title":"Updated","is_public":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/abc", strings.NewReader(`{"title":"Test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "abc")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpdate_InvalidBody(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpdate_NotFound_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		updateFn: func(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error) {
			return nil, domain.ErrNotFound
		},
	})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/99", strings.NewReader(`{"title":"Test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "99")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestUpdate_Forbidden_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		updateFn: func(ctx context.Context, userID, notebookID int64, title string, isPublic bool) (*domain.Notebook, error) {
			return nil, domain.ErrForbidden
		},
	})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1", strings.NewReader(`{"title":"Test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestUpdateBlock_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		updateBlockFn: func(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
			return &domain.Block{ID: blockID, NotebookID: notebookID, Type: cellType, Language: language, Content: content}, nil
		},
	})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/blocks/2",
		strings.NewReader(`{"type":"code","language":"python","content":"print(1)"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.UpdateBlock(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestUpdateBlock_InvalidNotebookID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/abc/blocks/2", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "abc")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.UpdateBlock(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpdateBlock_InvalidBlockID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/blocks/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "abc")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.UpdateBlock(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpdateBlock_InvalidBody(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/blocks/2", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.UpdateBlock(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpdateBlock_NotFound_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		updateBlockFn: func(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
			return nil, domain.ErrNotFound
		},
	})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/blocks/2",
		strings.NewReader(`{"type":"code","language":"python","content":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.UpdateBlock(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestUpdateBlock_Forbidden_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		updateBlockFn: func(ctx context.Context, userID, notebookID, blockID int64, content, cellType, language string) (*domain.Block, error) {
			return nil, domain.ErrForbidden
		},
	})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/blocks/2",
		strings.NewReader(`{"type":"code","language":"python","content":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.UpdateBlock(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestDeleteBlock_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		deleteBlockFn: func(ctx context.Context, userID, notebookID, blockID int64) error {
			return nil
		},
	})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/blocks/2", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.DeleteBlock(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d", rec.Code)
	}
}

func TestDeleteBlock_InvalidNotebookID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/abc/blocks/2", nil)
	req.SetPathValue("id", "abc")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.DeleteBlock(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestDeleteBlock_InvalidBlockID(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/blocks/abc", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "abc")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.DeleteBlock(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestDeleteBlock_NotFound_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		deleteBlockFn: func(ctx context.Context, userID, notebookID, blockID int64) error {
			return domain.ErrNotFound
		},
	})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/blocks/2", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.DeleteBlock(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestDeleteBlock_Forbidden_Handler(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		deleteBlockFn: func(ctx context.Context, userID, notebookID, blockID int64) error {
			return domain.ErrForbidden
		},
	})
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/blocks/2", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "2")
	req = reqWithUser(req, testUser)
	rec := httptest.NewRecorder()
	h.DeleteBlock(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}
