package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/dto"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

type AdminHandler struct {
	authClient     pbauth.AuthServiceClient
	notebookClient pbnotebook.NotebookServiceClient
}

func NewAdminHandler(authClient pbauth.AuthServiceClient, notebookClient pbnotebook.NotebookServiceClient) *AdminHandler {
	return &AdminHandler{authClient: authClient, notebookClient: notebookClient}
}

func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux, authMw, adminMw middleware.Middleware) {
	chain := func(handler http.HandlerFunc) http.Handler {
		return authMw(adminMw(http.HandlerFunc(handler)))
	}
	mux.Handle("GET /api/v1/admin/users", chain(h.ListUsers))
	mux.Handle("POST /api/v1/admin/users/{id}/ban", chain(h.BanUser))
	mux.Handle("POST /api/v1/admin/users/{id}/unban", chain(h.UnbanUser))
	mux.Handle("GET /api/v1/admin/notebooks", chain(h.ListNotebooks))
	mux.Handle("DELETE /api/v1/admin/notebooks/{id}", chain(h.DeleteNotebook))
	mux.Handle("GET /api/v1/admin/stats", chain(h.GetStats))
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	search := r.URL.Query().Get("search")

	resp, err := h.authClient.AdminListUsers(r.Context(), &pbauth.AdminListUsersRequest{
		AdminUserId: user.ID,
		Limit:       int32(limit),  //nolint:gosec
		Offset:      int32(offset), //nolint:gosec
		Search:      search,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	users := make([]dto.UserResponse, len(resp.GetUsers()))
	for i, u := range resp.GetUsers() {
		users[i] = protoUserToDTO(u)
	}
	httputil.JSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": resp.GetTotal(),
	})
}

func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	_, err = h.authClient.AdminSetBan(r.Context(), &pbauth.AdminSetBanRequest{
		AdminUserId:  user.ID,
		TargetUserId: targetID,
		Ban:          true,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, nil)
}

func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	_, err = h.authClient.AdminSetBan(r.Context(), &pbauth.AdminSetBanRequest{
		AdminUserId:  user.ID,
		TargetUserId: targetID,
		Ban:          false,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, nil)
}

func (h *AdminHandler) ListNotebooks(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	search := r.URL.Query().Get("search")

	resp, err := h.notebookClient.AdminListNotebooks(r.Context(), &pbnotebook.AdminListNotebooksRequest{
		Limit:  int32(limit),  //nolint:gosec
		Offset: int32(offset), //nolint:gosec
		Search: search,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	type notebookItem struct {
		ID        int64     `json:"id"`
		OwnerID   int64     `json:"owner_id"`
		Title     string    `json:"title"`
		IsPublic  bool      `json:"is_public"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	notebooks := make([]notebookItem, len(resp.GetNotebooks()))
	for i, nb := range resp.GetNotebooks() {
		notebooks[i] = notebookItem{
			ID:        nb.GetId(),
			OwnerID:   nb.GetOwnerId(),
			Title:     nb.GetTitle(),
			IsPublic:  nb.GetIsPublic(),
			CreatedAt: time.Unix(nb.GetCreatedAt(), 0),
			UpdatedAt: time.Unix(nb.GetUpdatedAt(), 0),
		}
	}
	httputil.JSON(w, http.StatusOK, map[string]interface{}{
		"notebooks": notebooks,
		"total":     resp.GetTotal(),
	})
}

func (h *AdminHandler) DeleteNotebook(w http.ResponseWriter, r *http.Request) {
	notebookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	_, err = h.notebookClient.AdminDeleteNotebook(r.Context(), &pbnotebook.AdminDeleteNotebookRequest{
		NotebookId: notebookID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, nil)
}

func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	resp, err := h.authClient.AdminGetStats(r.Context(), &pbauth.AdminGetStatsRequest{
		AdminUserId: user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{
		"total_users":    resp.GetTotalUsers(),
		"total_sessions": resp.GetTotalSessions(),
		"dau":            resp.GetDau(),
		"mau":            resp.GetMau(),
	})
}
