package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/issue"
)

type IssueHandler struct {
	client     pb.IssueServiceClient
	authClient pbauth.AuthServiceClient
}

func NewIssueHandler(client pb.IssueServiceClient, authClient pbauth.AuthServiceClient) *IssueHandler {
	return &IssueHandler{client: client, authClient: authClient}
}

func (h *IssueHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware, adminMw middleware.Middleware) {
	mux.Handle("GET /api/v1/issues", authMw(http.HandlerFunc(h.GetAll)))
	mux.Handle("GET /api/v1/issues/{id}", authMw(http.HandlerFunc(h.GetByID)))
	mux.Handle("POST /api/v1/issues", authMw(http.HandlerFunc(h.Create)))
	mux.Handle("DELETE /api/v1/issues/{id}", authMw(http.HandlerFunc(h.Delete)))
	mux.Handle("POST /api/v1/issues/{id}/messages", authMw(http.HandlerFunc(h.AddMessage)))

	mux.Handle("GET /api/v1/admin/issues/stats", authMw(adminMw(http.HandlerFunc(h.AdminGetStats))))
	mux.Handle("GET /api/v1/admin/issues/{id}", authMw(adminMw(http.HandlerFunc(h.AdminGetByID))))
	mux.Handle("GET /api/v1/admin/issues", authMw(adminMw(http.HandlerFunc(h.AdminGetAll))))
	mux.Handle("PATCH /api/v1/admin/issues/{id}/status", authMw(adminMw(http.HandlerFunc(h.AdminUpdateStatus))))
	mux.Handle("POST /api/v1/admin/issues/{id}/response", authMw(adminMw(http.HandlerFunc(h.AdminAddResponse))))
}

// RegisterRoues is kept for backwards compatibility with gateway app.go.
// Deprecated: use RegisterRoutes instead.
func (h *IssueHandler) RegisterRoues(mux *http.ServeMux, authMw middleware.Middleware) {
	h.RegisterRoutes(mux, authMw, func(next http.Handler) http.Handler { return next })
}

// ── request / response types ────────────────────────────────────────────────

type createIssueRequest struct {
	Category string `json:"category"`
	Content  string `json:"content"`
}

type addMessageRequest struct {
	Content string `json:"content"`
}

type patchIssueStatusRequest struct {
	Status string `json:"status"`
}

type adminIssueResponseRequest struct {
	Content string `json:"content"`
}

type issueMessageResponse struct {
	ID        int64     `json:"id"`
	IssueID   int64     `json:"issue_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	IsAdmin   bool      `json:"is_admin"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type issueResponse struct {
	ID        int64                  `json:"id"`
	UserID    int64                  `json:"user_id"`
	Category  string                 `json:"category"`
	Status    string                 `json:"status"`
	Content   string                 `json:"content"`
	Messages  []issueMessageResponse `json:"messages"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type adminIssueResponse struct {
	ID        int64                  `json:"id"`
	UserID    int64                  `json:"user_id"`
	Username  string                 `json:"username,omitempty"`
	Category  string                 `json:"category"`
	Status    string                 `json:"status"`
	Content   string                 `json:"content"`
	Messages  []issueMessageResponse `json:"messages"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type issueListResponse struct {
	Issues []issueResponse `json:"issues"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type adminIssueListResponse struct {
	Issues []adminIssueResponse `json:"issues"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

type issueStatsResponse struct {
	Total      int64              `json:"total"`
	Open       int64              `json:"open"`
	InProgress int64              `json:"in_progress"`
	Closed     int64              `json:"closed"`
	ByCategory issueCategoryStats `json:"by_category"`
}

type issueCategoryStats struct {
	Bug      int64 `json:"bug"`
	Idea     int64 `json:"idea"`
	Problem  int64 `json:"problem"`
	Feedback int64 `json:"feedback"`
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (h *IssueHandler) paginationParams(r *http.Request) (limit, offset int) {
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	return
}

func protoIssueToResponse(issue *pb.IssueInfo, msgs []*pb.IssueMessageInfo) issueResponse {
	if issue == nil {
		return issueResponse{}
	}
	resp := issueResponse{
		ID:        issue.GetId(),
		UserID:    issue.GetUserId(),
		Category:  issue.GetCategory(),
		Status:    issue.GetStatus(),
		Content:   issue.GetContent(),
		Messages:  make([]issueMessageResponse, 0),
		CreatedAt: time.Unix(issue.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(issue.GetUpdatedAt(), 0),
	}
	for _, m := range msgs {
		resp.Messages = append(resp.Messages, protoMsgToResponse(m))
	}
	return resp
}

func protoIssueToAdminResponse(issue *pb.IssueInfo, msgs []*pb.IssueMessageInfo) adminIssueResponse {
	if issue == nil {
		return adminIssueResponse{}
	}
	resp := adminIssueResponse{
		ID:        issue.GetId(),
		UserID:    issue.GetUserId(),
		Category:  issue.GetCategory(),
		Status:    issue.GetStatus(),
		Content:   issue.GetContent(),
		Messages:  make([]issueMessageResponse, 0),
		CreatedAt: time.Unix(issue.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(issue.GetUpdatedAt(), 0),
	}
	for _, m := range msgs {
		resp.Messages = append(resp.Messages, protoMsgToResponse(m))
	}
	return resp
}

func protoMsgToResponse(m *pb.IssueMessageInfo) issueMessageResponse {
	if m == nil {
		return issueMessageResponse{}
	}
	return issueMessageResponse{
		ID:        m.GetId(),
		IssueID:   m.GetIssueId(),
		UserID:    m.GetUserId(),
		IsAdmin:   m.GetIsAdmin(),
		Content:   m.GetContent(),
		CreatedAt: time.Unix(m.GetCreatedAt(), 0),
	}
}

func (h *IssueHandler) enrichUsername(r *http.Request, userID int64) string {
	if h.authClient == nil {
		return ""
	}
	resp, err := h.authClient.GetUserByID(r.Context(), &pbauth.GetUserByIDRequest{UserId: userID})
	if err != nil {
		return ""
	}
	return resp.GetUser().GetUsername()
}

// ── user endpoints ───────────────────────────────────────────────────────────

func (h *IssueHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, offset := h.paginationParams(r)
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")

	resp, err := h.client.GetAll(r.Context(), &pb.GetAllIssuesRequest{
		Limit:    int32(limit),
		Offset:   int32(offset),
		Category: category,
		Status:   status,
		UserId:   user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	items := make([]issueResponse, len(resp.GetIssues()))
	for i, issue := range resp.GetIssues() {
		items[i] = protoIssueToResponse(issue, nil)
	}

	httputil.JSON(w, http.StatusOK, issueListResponse{
		Issues: items,
		Total:  int(resp.GetTotal()),
		Limit:  int(resp.GetLimit()),
		Offset: int(resp.GetOffset()),
	})
}

func (h *IssueHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	resp, err := h.client.GetByID(r.Context(), &pb.GetIssueRequest{Id: id})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	if resp.GetIssue().GetUserId() != user.ID {
		httputil.Error(w, http.StatusForbidden, "access denied")
		return
	}

	httputil.JSON(w, http.StatusOK, protoIssueToResponse(resp.GetIssue(), resp.GetMessages()))
}

func (h *IssueHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createIssueRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Category == "" {
		httputil.Error(w, http.StatusBadRequest, "category is required")
		return
	}
	if req.Content == "" {
		httputil.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	resp, err := h.client.Create(r.Context(), &pb.CreateIssueRequest{
		Category: req.Category,
		Content:  req.Content,
		UserId:   user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, protoIssueToResponse(resp.GetIssue(), nil))
}

func (h *IssueHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	_, err = h.client.Delete(r.Context(), &pb.DeleteIssueRequest{
		Id:     id,
		UserId: user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

func (h *IssueHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	// Verify ownership before adding message.
	issueResp, err := h.client.GetByID(r.Context(), &pb.GetIssueRequest{Id: id})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	if issueResp.GetIssue().GetUserId() != user.ID {
		httputil.Error(w, http.StatusForbidden, "access denied")
		return
	}

	var req addMessageRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Content == "" {
		httputil.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	msgResp, err := h.client.AddMessage(r.Context(), &pb.AddIssueMessageRequest{
		IssueId: id,
		UserId:  user.ID,
		IsAdmin: false,
		Content: req.Content,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	m := protoMsgToResponse(msgResp.GetMessage())
	m.Username = user.Username
	httputil.JSON(w, http.StatusCreated, m)
}

// ── admin endpoints ──────────────────────────────────────────────────────────

func (h *IssueHandler) AdminGetAll(w http.ResponseWriter, r *http.Request) {
	limit, offset := h.paginationParams(r)
	statusFilter := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")
	q := r.URL.Query().Get("q")

	var userID int64
	if raw := r.URL.Query().Get("userid"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			userID = parsed
		}
	}

	resp, err := h.client.AdminGetAllIssues(r.Context(), &pb.AdminGetAllIssuesRequest{
		Limit:    int32(limit),
		Offset:   int32(offset),
		Category: category,
		Status:   statusFilter,
		UserId:   userID,
		Content:  q,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	items := make([]adminIssueResponse, len(resp.GetIssues()))
	for i, issue := range resp.GetIssues() {
		items[i] = protoIssueToAdminResponse(issue, nil)
		items[i].Username = h.enrichUsername(r, issue.GetUserId())
	}

	httputil.JSON(w, http.StatusOK, adminIssueListResponse{
		Issues: items,
		Total:  int(resp.GetTotal()),
		Limit:  int(resp.GetLimit()),
		Offset: int(resp.GetOffset()),
	})
}

func (h *IssueHandler) AdminGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	resp, err := h.client.GetByID(r.Context(), &pb.GetIssueRequest{Id: id})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	item := protoIssueToAdminResponse(resp.GetIssue(), resp.GetMessages())
	item.Username = h.enrichUsername(r, resp.GetIssue().GetUserId())

	// Enrich message usernames.
	for i, m := range item.Messages {
		item.Messages[i].Username = h.enrichUsername(r, m.UserID)
	}

	httputil.JSON(w, http.StatusOK, item)
}

func (h *IssueHandler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	var req patchIssueStatusRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status == "" {
		httputil.Error(w, http.StatusBadRequest, "status is required")
		return
	}

	resp, err := h.client.AdminUpdateStatus(r.Context(), &pb.AdminUpdateIssueStatusRequest{
		Id:     id,
		Status: req.Status,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	item := protoIssueToAdminResponse(resp.GetIssue(), resp.GetMessages())
	item.Username = h.enrichUsername(r, resp.GetIssue().GetUserId())
	httputil.JSON(w, http.StatusOK, item)
}

func (h *IssueHandler) AdminAddResponse(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	var req adminIssueResponseRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Content == "" {
		httputil.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	msgResp, err := h.client.AddMessage(r.Context(), &pb.AddIssueMessageRequest{
		IssueId: id,
		UserId:  user.ID,
		IsAdmin: true,
		Content: req.Content,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	m := protoMsgToResponse(msgResp.GetMessage())
	m.Username = user.Username
	httputil.JSON(w, http.StatusCreated, m)
}

func (h *IssueHandler) AdminGetStats(w http.ResponseWriter, r *http.Request) {
	resp, err := h.client.GetStats(r.Context(), &pb.GetIssueStatsRequest{})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, issueStatsResponse{
		Total:      resp.GetTotal(),
		Open:       resp.GetOpen(),
		InProgress: resp.GetInProgress(),
		Closed:     resp.GetClosed(),
		ByCategory: issueCategoryStats{
			Bug:      resp.GetBug(),
			Idea:     resp.GetIdea(),
			Problem:  resp.GetProblem(),
			Feedback: resp.GetFeedback(),
		},
	})
}
