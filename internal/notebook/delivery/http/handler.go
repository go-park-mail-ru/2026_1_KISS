package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
)

type notebookUsecase interface {
	Create(ctx context.Context, userID int64, title string) (*domain.Notebook, error)
	GetByID(ctx context.Context, userID, notebookID int64) (*domain.Notebook, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, error)
	Delete(ctx context.Context, userID, notebookID int64) error
	AddBlock(ctx context.Context, userID, notebookID int64, block *domain.Block) (*domain.Block, error)
}

type NotebookHandler struct {
	usecase notebookUsecase
}

func New(uc notebookUsecase) *NotebookHandler {
	return &NotebookHandler{usecase: uc}
}

func (h *NotebookHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("GET /api/v1/notebooks", authMw(http.HandlerFunc(h.List)))
	mux.Handle("POST /api/v1/notebooks", authMw(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/notebooks/{id}", authMw(http.HandlerFunc(h.GetByID)))
	mux.Handle("DELETE /api/v1/notebooks/{id}", authMw(http.HandlerFunc(h.Delete)))
	mux.Handle("POST /api/v1/notebooks/{id}/blocks", authMw(http.HandlerFunc(h.AddBlock)))
}

func (h *NotebookHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	notebooks, err := h.usecase.ListByUser(r.Context(), user.ID, limit, offset)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, NewNotebookListResponse(notebooks))
}

func (h *NotebookHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var req CreateNotebookRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	nb, err := h.usecase.Create(r.Context(), user.ID, req.Title)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, NewNotebookResponse(nb))
}

func (h *NotebookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	id, err := parseID(r)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	nb, err := h.usecase.GetByID(r.Context(), user.ID, id)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, NewNotebookResponse(nb))
}

func (h *NotebookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	id, err := parseID(r)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	if err := h.usecase.Delete(r.Context(), user.ID, id); err != nil {
		mapDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NotebookHandler) AddBlock(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	id, err := parseID(r)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	var req CreateBlockRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	block := &domain.Block{
		Type:     req.Type,
		Language: req.Language,
		Content:  req.Content,
	}

	created, err := h.usecase.AddBlock(r.Context(), user.ID, id, block)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	resp := BlockResponse{
		ID:        created.ID,
		Type:      created.Type,
		Language:  created.Language,
		Content:   created.Content,
		Position:  created.Position,
		CreatedAt: created.CreatedAt,
	}
	httputil.JSON(w, http.StatusCreated, resp)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func mapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		httputil.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		httputil.Error(w, http.StatusConflict, "conflict")
	case errors.Is(err, domain.ErrUnauthorized):
		httputil.Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, domain.ErrInvalidInput):
		httputil.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		httputil.Error(w, http.StatusForbidden, "access denied")
	default:
		httputil.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
