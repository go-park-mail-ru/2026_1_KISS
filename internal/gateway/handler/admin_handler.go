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
	pbstorage "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type AdminHandler struct {
	authClient     pbauth.AuthServiceClient
	notebookClient pbnotebook.NotebookServiceClient
	storageClient  pbstorage.StorageServiceClient
}

func NewAdminHandler(authClient pbauth.AuthServiceClient, notebookClient pbnotebook.NotebookServiceClient, storageClient pbstorage.StorageServiceClient) *AdminHandler {
	return &AdminHandler{authClient: authClient, notebookClient: notebookClient, storageClient: storageClient}
}

func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux, authMw, adminMw middleware.Middleware) {
	chain := func(handler http.HandlerFunc) http.Handler {
		return authMw(adminMw(http.HandlerFunc(handler)))
	}
	mux.Handle("GET /api/v1/admin/users", chain(h.ListUsers))
	mux.Handle("POST /api/v1/admin/users/{id}/ban", chain(h.BanUser))
	mux.Handle("POST /api/v1/admin/users/{id}/unban", chain(h.UnbanUser))
	mux.Handle("PUT /api/v1/admin/users/{id}", chain(h.UpdateUser))
	mux.Handle("PUT /api/v1/admin/users/{id}/password", chain(h.ResetPassword))
	mux.Handle("PUT /api/v1/admin/users/{id}/plan", chain(h.SetPlan))
	mux.Handle("GET /api/v1/admin/notebooks", chain(h.ListNotebooks))
	mux.Handle("DELETE /api/v1/admin/notebooks/{id}", chain(h.DeleteNotebook))
	mux.Handle("GET /api/v1/admin/stats", chain(h.GetStats))
	mux.Handle("GET /api/v1/admin/stats/activity", chain(h.GetActivityStats))
	mux.Handle("GET /api/v1/admin/storage/stats", chain(h.GetStorageStats))
	mux.Handle("GET /api/v1/admin/storage/files", chain(h.ListStorageFiles))
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	search := r.URL.Query().Get("search")

	verifiedParam := r.URL.Query().Get("verified")
	req := &pbauth.AdminListUsersRequest{
		AdminUserId: user.ID,
		Limit:       int32(limit),  //nolint:gosec
		Offset:      int32(offset), //nolint:gosec
		Search:      search,
	}
	if verifiedParam == "true" {
		v := true
		req.Verified = &v
	} else if verifiedParam == "false" {
		v := false
		req.Verified = &v
	}

	resp, err := h.authClient.AdminListUsers(r.Context(), req)
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	users := make([]dto.UserResponse, len(resp.GetUsers()))
	for i, u := range resp.GetUsers() {
		users[i] = protoUserToDTO(u)
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
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

	_, _ = h.notebookClient.AdminSetUserNotebooksPrivate(r.Context(), &pbnotebook.AdminSetUserNotebooksPrivateRequest{
		OwnerId: targetID,
	})

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

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authClient.AdminUpdateUser(r.Context(), &pbauth.AdminUpdateUserRequest{
		AdminUserId:  user.ID,
		TargetUserId: targetID,
		Username:     req.Username,
		Email:        req.Email,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, protoUserToDTO(resp.GetUser()))
}

func (h *AdminHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = h.authClient.AdminResetPassword(r.Context(), &pbauth.AdminResetPasswordRequest{
		AdminUserId:  user.ID,
		TargetUserId: targetID,
		NewPassword:  req.Password,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, nil)
}

func (h *AdminHandler) SetPlan(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = h.authClient.AdminSetPlan(r.Context(), &pbauth.AdminSetPlanRequest{
		AdminUserId:  user.ID,
		TargetUserId: targetID,
		Plan:         req.Plan,
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
	httputil.JSON(w, http.StatusOK, map[string]any{
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

	nbResp, nbErr := h.notebookClient.AdminGetNotebookCount(r.Context(), &pbnotebook.AdminGetNotebookCountRequest{})
	var totalNotebooks int64
	if nbErr == nil {
		totalNotebooks = nbResp.GetTotal()
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"total_users":     resp.GetTotalUsers(),
		"total_sessions":  resp.GetTotalSessions(),
		"dau":             resp.GetDau(),
		"mau":             resp.GetMau(),
		"total_notebooks": totalNotebooks,
	})
}

func (h *AdminHandler) GetActivityStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	dauDays, _ := strconv.Atoi(r.URL.Query().Get("dau_days"))
	mauMonths, _ := strconv.Atoi(r.URL.Query().Get("mau_months"))

	resp, err := h.authClient.AdminGetActivityStats(r.Context(), &pbauth.AdminGetActivityStatsRequest{
		AdminUserId: user.ID,
		DauDays:     int32(dauDays),   //nolint:gosec
		MauMonths:   int32(mauMonths), //nolint:gosec
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	type dauItem struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	type mauItem struct {
		Month string `json:"month"`
		Count int64  `json:"count"`
	}

	dau := make([]dauItem, len(resp.GetDau()))
	for i, d := range resp.GetDau() {
		dau[i] = dauItem{Date: d.GetDate(), Count: d.GetCount()}
	}
	mau := make([]mauItem, len(resp.GetMau()))
	for i, m := range resp.GetMau() {
		mau[i] = mauItem{Month: m.GetMonth(), Count: m.GetCount()}
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"dau": dau,
		"mau": mau,
	})
}

func (h *AdminHandler) GetStorageStats(w http.ResponseWriter, r *http.Request) {
	resp, err := h.storageClient.GetStorageStats(r.Context(), &pbstorage.GetStorageStatsRequest{})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"total_files":       resp.GetTotalFiles(),
		"total_size_bytes":  resp.GetTotalSizeBytes(),
		"files_by_category": resp.GetFilesByCategory(),
		"size_by_category":  resp.GetSizeByCategory(),
	})
}

func (h *AdminHandler) ListStorageFiles(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	ownerID, _ := strconv.ParseInt(r.URL.Query().Get("owner_id"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.storageClient.AdminListFiles(r.Context(), &pbstorage.AdminListFilesRequest{
		Category: category,
		OwnerId:  ownerID,
		Limit:    int32(limit),  //nolint:gosec
		Offset:   int32(offset), //nolint:gosec
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	files := make([]fileResponse, len(resp.GetFiles()))
	for i, f := range resp.GetFiles() {
		files[i] = fileInfoToResponse(f)
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"files": files,
		"total": resp.GetTotal(),
	})
}
