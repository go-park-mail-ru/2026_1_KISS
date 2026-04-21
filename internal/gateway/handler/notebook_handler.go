package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

type NotebookHandler struct {
	client     pb.NotebookServiceClient
	authClient pbauth.AuthServiceClient
}

func NewNotebookHandler(client pb.NotebookServiceClient, authClient pbauth.AuthServiceClient) *NotebookHandler {
	return &NotebookHandler{client: client, authClient: authClient}
}

func (h *NotebookHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("GET /api/v1/notebooks", authMw(http.HandlerFunc(h.List)))
	mux.Handle("POST /api/v1/notebooks", authMw(http.HandlerFunc(h.Create)))
	// Literal segment "shared" has priority over {id}, so register first.
	mux.Handle("GET /api/v1/notebooks/shared", authMw(http.HandlerFunc(h.ListShared)))
	mux.Handle("GET /api/v1/notebooks/{id}", authMw(http.HandlerFunc(h.GetByID)))
	mux.Handle("PUT /api/v1/notebooks/{id}", authMw(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /api/v1/notebooks/{id}", authMw(http.HandlerFunc(h.Delete)))
	mux.Handle("POST /api/v1/notebooks/{id}/blocks", authMw(http.HandlerFunc(h.AddBlock)))
	mux.Handle("PUT /api/v1/notebooks/{id}/blocks/{blockID}", authMw(http.HandlerFunc(h.UpdateBlock)))
	mux.Handle("DELETE /api/v1/notebooks/{id}/blocks/{blockID}", authMw(http.HandlerFunc(h.DeleteBlock)))
	mux.Handle("GET /api/v1/notebooks/{id}/permissions", authMw(http.HandlerFunc(h.ListPermissions)))
	mux.Handle("POST /api/v1/notebooks/{id}/permissions/invite", authMw(http.HandlerFunc(h.GrantPermissionByIdentifier)))
	mux.Handle("PUT /api/v1/notebooks/{id}/permissions/{userID}", authMw(http.HandlerFunc(h.GrantPermission)))
	mux.Handle("DELETE /api/v1/notebooks/{id}/permissions/{userID}", authMw(http.HandlerFunc(h.RevokePermission)))
}

type createNotebookRequest struct {
	Title string `json:"title"`
}

type updateNotebookRequest struct {
	Title    string `json:"title"`
	IsPublic bool   `json:"is_public"`
}

type createBlockRequest struct {
	Type     string `json:"type"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

type updateBlockRequest struct {
	Type     string `json:"type"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

type notebookResponse struct {
	ID        int64           `json:"id"`
	OwnerID   int64           `json:"owner_id"`
	Title     string          `json:"title"`
	IsPublic  bool            `json:"is_public"`
	Blocks    []blockResponse `json:"blocks,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type blockResponse struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Language  string    `json:"language"`
	Content   string    `json:"content"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

type notebookListResponse struct {
	Notebooks []notebookResponse `json:"notebooks"`
	Total     int                `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}

func (h *NotebookHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	search := r.URL.Query().Get("search")

	resp, err := h.client.ListByUser(r.Context(), &pb.ListNotebooksRequest{
		UserId: user.ID,
		Limit:  int32(limit),  //nolint:gosec // pagination limit fits int32
		Offset: int32(offset), //nolint:gosec // pagination offset fits int32
		Search: search,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	items := make([]notebookResponse, len(resp.GetNotebooks()))
	for i, nb := range resp.GetNotebooks() {
		items[i] = protoNotebookToResponse(nb)
	}

	httputil.JSON(w, http.StatusOK, notebookListResponse{
		Notebooks: items,
		Total:     int(resp.GetTotal()),
		Limit:     int(resp.GetLimit()),
		Offset:    int(resp.GetOffset()),
	})
}

func (h *NotebookHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createNotebookRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.client.Create(r.Context(), &pb.CreateNotebookRequest{
		UserId: user.ID,
		Title:  req.Title,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, protoNotebookToResponse(resp.GetNotebook()))
}

func (h *NotebookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	resp, err := h.client.GetByID(r.Context(), &pb.GetNotebookRequest{
		UserId:     user.ID,
		NotebookId: id,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoNotebookToResponse(resp.GetNotebook()))
}

func (h *NotebookHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	var req updateNotebookRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.client.Update(r.Context(), &pb.UpdateNotebookRequest{
		UserId:     user.ID,
		NotebookId: id,
		Title:      req.Title,
		IsPublic:   req.IsPublic,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoNotebookToResponse(resp.GetNotebook()))
}

func (h *NotebookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	_, err = h.client.Delete(r.Context(), &pb.DeleteNotebookRequest{
		UserId:     user.ID,
		NotebookId: id,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

func (h *NotebookHandler) AddBlock(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	var req createBlockRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.client.AddBlock(r.Context(), &pb.AddBlockRequest{
		UserId:     user.ID,
		NotebookId: id,
		Type:       req.Type,
		Language:   req.Language,
		Content:    req.Content,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, protoBlockToResponse(resp.GetBlock()))
}

func (h *NotebookHandler) UpdateBlock(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	blockID, err := strconv.ParseInt(r.PathValue("blockID"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid block id")
		return
	}

	var req updateBlockRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.client.UpdateBlock(r.Context(), &pb.UpdateBlockRequest{
		UserId:     user.ID,
		NotebookId: id,
		BlockId:    blockID,
		Content:    req.Content,
		Type:       req.Type,
		Language:   req.Language,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoBlockToResponse(resp.GetBlock()))
}

func (h *NotebookHandler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	blockID, err := strconv.ParseInt(r.PathValue("blockID"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid block id")
		return
	}

	_, err = h.client.DeleteBlock(r.Context(), &pb.DeleteBlockRequest{
		UserId:     user.ID,
		NotebookId: id,
		BlockId:    blockID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

type grantPermissionRequest struct {
	Level string `json:"level"`
}

type grantPermissionByIdentifierRequest struct {
	Identifier string `json:"identifier"`
	Level      string `json:"level"`
}

type permissionResponse struct {
	NotebookID      int64  `json:"notebook_id"`
	UserID          int64  `json:"user_id"`
	PermissionLevel string `json:"permission_level"`
}

type permissionListResponse struct {
	Permissions []permissionResponse `json:"permissions"`
}

func (h *NotebookHandler) ListShared(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.client.ListSharedWithUser(r.Context(), &pb.ListSharedWithUserRequest{
		UserId: user.ID,
		Limit:  int32(limit),  //nolint:gosec // pagination limit fits int32
		Offset: int32(offset), //nolint:gosec // pagination offset fits int32
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	items := make([]notebookResponse, len(resp.GetNotebooks()))
	for i, nb := range resp.GetNotebooks() {
		items[i] = protoNotebookToResponse(nb)
	}

	httputil.JSON(w, http.StatusOK, notebookListResponse{
		Notebooks: items,
		Total:     int(resp.GetTotal()),
		Limit:     int(resp.GetLimit()),
		Offset:    int(resp.GetOffset()),
	})
}

func (h *NotebookHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	resp, err := h.client.ListPermissions(r.Context(), &pb.ListPermissionsRequest{
		RequesterId: user.ID,
		NotebookId:  id,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	items := make([]permissionResponse, len(resp.GetPermissions()))
	for i, p := range resp.GetPermissions() {
		items[i] = permissionResponse{
			NotebookID:      p.GetNotebookId(),
			UserID:          p.GetUserId(),
			PermissionLevel: p.GetPermissionLevel(),
		}
	}

	httputil.JSON(w, http.StatusOK, permissionListResponse{Permissions: items})
}

func (h *NotebookHandler) GrantPermissionByIdentifier(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notebookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	var req grantPermissionByIdentifierRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Identifier == "" {
		httputil.Error(w, http.StatusBadRequest, "identifier is required")
		return
	}
	if req.Level == "" {
		httputil.Error(w, http.StatusBadRequest, "level is required")
		return
	}

	authResp, err := h.authClient.GetUserByIdentifier(r.Context(), &pbauth.GetUserByIdentifierRequest{
		Identifier: req.Identifier,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	targetUserID := authResp.GetUser().GetId()

	_, err = h.client.GrantPermission(r.Context(), &pb.GrantPermissionRequest{
		RequesterId:  user.ID,
		NotebookId:   notebookID,
		TargetUserId: targetUserID,
		Level:        req.Level,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, permissionResponse{
		NotebookID:      notebookID,
		UserID:          targetUserID,
		PermissionLevel: req.Level,
	})
}

func (h *NotebookHandler) GrantPermission(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	targetUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req grantPermissionRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = h.client.GrantPermission(r.Context(), &pb.GrantPermissionRequest{
		RequesterId:  user.ID,
		NotebookId:   id,
		TargetUserId: targetUserID,
		Level:        req.Level,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, permissionResponse{
		NotebookID:      id,
		UserID:          targetUserID,
		PermissionLevel: req.Level,
	})
}

func (h *NotebookHandler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid notebook id")
		return
	}

	targetUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	_, err = h.client.RevokePermission(r.Context(), &pb.RevokePermissionRequest{
		RequesterId:  user.ID,
		NotebookId:   id,
		TargetUserId: targetUserID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func protoNotebookToResponse(nb *pb.NotebookInfo) notebookResponse {
	resp := notebookResponse{
		ID:        nb.GetId(),
		OwnerID:   nb.GetOwnerId(),
		Title:     nb.GetTitle(),
		IsPublic:  nb.GetIsPublic(),
		CreatedAt: time.Unix(nb.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(nb.GetUpdatedAt(), 0),
	}
	if len(nb.GetBlocks()) > 0 {
		resp.Blocks = make([]blockResponse, len(nb.GetBlocks()))
		for i, b := range nb.GetBlocks() {
			resp.Blocks[i] = protoBlockToResponse(b)
		}
	}
	return resp
}

func protoBlockToResponse(b *pb.BlockInfo) blockResponse {
	return blockResponse{
		ID:        b.GetId(),
		Type:      b.GetType(),
		Language:  b.GetLanguage(),
		Content:   b.GetContent(),
		Position:  int(b.GetPosition()),
		CreatedAt: time.Unix(b.GetCreatedAt(), 0),
	}
}
