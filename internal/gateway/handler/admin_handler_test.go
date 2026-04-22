package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

func TestAdminHandler_ListUsers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().AdminListUsers(gomock.Any(), gomock.Any()).Return(&pbauth.AdminListUsersResponse{
		Users: []*pbauth.UserInfo{
			{Id: 1, Username: "admin", Email: "admin@test.com", IsAdmin: true},
			{Id: 2, Username: "user", Email: "user@test.com"},
		},
		Total: 2,
	}, nil)

	h := NewAdminHandler(authClient, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/users?limit=10&offset=0&search=test", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_ListUsers_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().AdminListUsers(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "internal"))

	h := NewAdminHandler(authClient, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListUsers(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestAdminHandler_BanUser_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().AdminSetBan(gomock.Any(), gomock.Any()).
		Return(&pbauth.AdminSetBanResponse{}, nil)

	h := NewAdminHandler(authClient, nil)
	req := httptest.NewRequest("POST", "/api/v1/admin/users/2/ban", nil)
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.BanUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_BanUser_InvalidID(t *testing.T) {
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/admin/users/abc/ban", nil)
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.BanUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_BanUser_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().AdminSetBan(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewAdminHandler(authClient, nil)
	req := httptest.NewRequest("POST", "/api/v1/admin/users/2/ban", nil)
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.BanUser(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestAdminHandler_UnbanUser_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().AdminSetBan(gomock.Any(), gomock.Any()).
		Return(&pbauth.AdminSetBanResponse{}, nil)

	h := NewAdminHandler(authClient, nil)
	req := httptest.NewRequest("POST", "/api/v1/admin/users/2/unban", nil)
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UnbanUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_UnbanUser_InvalidID(t *testing.T) {
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/admin/users/abc/unban", nil)
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UnbanUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_ListNotebooks_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)

	nbClient.EXPECT().AdminListNotebooks(gomock.Any(), gomock.Any()).Return(&pbnotebook.AdminListNotebooksResponse{
		Notebooks: []*pbnotebook.NotebookInfo{
			{Id: 1, OwnerId: 1, Title: "NB1", CreatedAt: 1000, UpdatedAt: 2000},
		},
		Total: 1,
	}, nil)

	h := NewAdminHandler(nil, nbClient)
	req := httptest.NewRequest("GET", "/api/v1/admin/notebooks?limit=10&offset=0", nil)
	rec := httptest.NewRecorder()

	h.ListNotebooks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_ListNotebooks_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)

	nbClient.EXPECT().AdminListNotebooks(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "internal"))

	h := NewAdminHandler(nil, nbClient)
	req := httptest.NewRequest("GET", "/api/v1/admin/notebooks", nil)
	rec := httptest.NewRecorder()

	h.ListNotebooks(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestAdminHandler_DeleteNotebook_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)

	nbClient.EXPECT().AdminDeleteNotebook(gomock.Any(), gomock.Any()).
		Return(&pbnotebook.DeleteNotebookResponse{}, nil)

	h := NewAdminHandler(nil, nbClient)
	req := httptest.NewRequest("DELETE", "/api/v1/admin/notebooks/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.DeleteNotebook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_DeleteNotebook_InvalidID(t *testing.T) {
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/admin/notebooks/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.DeleteNotebook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_DeleteNotebook_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)

	nbClient.EXPECT().AdminDeleteNotebook(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewAdminHandler(nil, nbClient)
	req := httptest.NewRequest("DELETE", "/api/v1/admin/notebooks/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.DeleteNotebook(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestAdminHandler_GetStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().AdminGetStats(gomock.Any(), gomock.Any()).Return(&pbauth.AdminGetStatsResponse{
		TotalUsers:    100,
		TotalSessions: 50,
		Dau:           10,
		Mau:           30,
	}, nil)

	h := NewAdminHandler(authClient, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_GetStats_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().AdminGetStats(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "internal"))

	h := NewAdminHandler(authClient, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetStats(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}
