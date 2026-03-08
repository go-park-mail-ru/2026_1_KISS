package http_test

import (
	"context"
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
	createFn     func(ctx context.Context, userID int64, title string) (*domain.Notebook, error)
	getByIDFn    func(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error)
	listByUserFn func(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, error)
	deleteFn     func(ctx context.Context, userID, notebookID int64) error
	addBlockFn   func(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error)
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

func (m *mockNotebookUsecase) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, limit, offset)
	}
	return []domain.Notebook{}, nil
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

func reqWithUser(req *http.Request, user *domain.User) *http.Request {
	ctx := middleware.SetUserInContext(req.Context(), user)
	return req.WithContext(ctx)
}

var testUser = &domain.User{ID: 1, Username: "testuser", Email: "test@example.com"}

func TestList(t *testing.T) {
	h := nbhttp.New(&mockNotebookUsecase{
		listByUserFn: func(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, error) {
			return []domain.Notebook{{ID: 1, Title: "Test"}}, nil
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
