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

func (h *IssueHandler) RegisterRoues(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("GET /api/v1/isstues", authMw(http.HandlerFunc(h.GetAll)))
	mux.Handle("GET /api/v1/issues/{id}", authMw(http.HandlerFunc(h.GetByID)))
	mux.Handle("POST /api/v1/issues", authMw(http.HandlerFunc(h.Create)))
	mux.Handle("PUT /api/v1/issues/{id}", authMw(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /api/v1/issues/{id}", authMw(http.HandlerFunc(h.Delete)))

	// Админские маршруты
	mux.Handle("GET /api/v1/admin/issues", authMw(http.HandlerFunc(h.AdminGetAllIssues)))

}

type createIssueRequest struct {
	Category string `json:"category"`
	Content  string `json:"content"`
}

type updateIssueRequest struct {
	Category string `json:"category"`
	Status   string `json:"status"`
	Content  string `json:"content"`
}

type issueResponse struct {
	ID        int64     `json:"id"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	Content   string    `json:"content"`
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type issueListResponse struct {
	Issues []issueResponse `json:"issues"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
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

	resp, err := h.client.GetByID(r.Context(), &pb.GetIssueRequest{
		Id: id,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoIssueToResponse(resp.GetIssue()))
}

func (h *IssueHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
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
		items[i] = protoIssueToResponse(issue)
		// Обогащаем email пользователя
		if h.authClient != nil {
			userResp, err := h.authClient.GetUserByID(r.Context(), &pbauth.GetUserByIDRequest{UserId: issue.GetUserId()})
			if err == nil {
				items[i].Email = userResp.GetUser().GetEmail()
			}
		}
	}

	httputil.JSON(w, http.StatusOK, issueListResponse{
		Issues: items,
		Total:  int(resp.GetTotal()),
		Limit:  int(resp.GetLimit()),
		Offset: int(resp.GetOffset()),
	})
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

	httputil.JSON(w, http.StatusCreated, protoIssueToResponse(resp.GetIssue()))
}

func (h *IssueHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req updateIssueRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.client.Update(r.Context(), &pb.UpdateIssueRequest{
		Id:       id,
		Category: req.Category,
		Status:   req.Status,
		Content:  req.Content,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, protoIssueToResponse(resp.GetIssue()))
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
		Id: id,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, nil)
}

func (h *IssueHandler) AdminGetAllIssues(w http.ResponseWriter, r *http.Request) {
	// Проверка прав администратора
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// TODO: проверка, что user является администратором
	// if !user.IsAdmin {
	//     httputil.Error(w, http.StatusForbidden, "access denied")
	//     return
	// }

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")

	resp, err := h.client.AdminGetAllIssues(r.Context(), &pb.AdminGetAllIssuesRequest{
		Limit:    int32(limit),
		Offset:   int32(offset),
		Category: category,
		Status:   status,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	items := make([]issueResponse, len(resp.GetIssues()))
	for i, issue := range resp.GetIssues() {
		items[i] = protoIssueToResponse(issue)
		// Обогащаем email пользователя
		if h.authClient != nil {
			userResp, err := h.authClient.GetUserByID(r.Context(), &pbauth.GetUserByIDRequest{UserId: issue.GetUserId()})
			if err == nil {
				items[i].Email = userResp.GetUser().GetEmail()
			}
		}
	}

	httputil.JSON(w, http.StatusOK, issueListResponse{
		Issues: items,
		Total:  int(resp.GetTotal()),
		Limit:  int(resp.GetLimit()),
		Offset: int(resp.GetOffset()),
	})
}

func protoIssueToResponse(issue *pb.IssueInfo) issueResponse {
	if issue == nil {
		return issueResponse{}
	}
	return issueResponse{
		ID:       issue.GetId(),
		Category: issue.GetCategory(),
		Status:   issue.GetStatus(),
		Content:  issue.GetContent(),
		UserID:   issue.GetUserId(),
	}
}
