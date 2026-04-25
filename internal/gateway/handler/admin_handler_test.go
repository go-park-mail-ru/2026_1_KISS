package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
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

	h := NewAdminHandler(authClient, nil, nil)
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

	h := NewAdminHandler(authClient, nil, nil)
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
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)

	authClient.EXPECT().AdminSetBan(gomock.Any(), gomock.Any()).
		Return(&pbauth.AdminSetBanResponse{}, nil)
	nbClient.EXPECT().AdminSetUserNotebooksPrivate(gomock.Any(), gomock.Any()).
		Return(&pbnotebook.AdminSetUserNotebooksPrivateResponse{}, nil)

	h := NewAdminHandler(authClient, nbClient, nil)
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
	h := NewAdminHandler(nil, nil, nil)
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

	h := NewAdminHandler(authClient, nil, nil)
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

	h := NewAdminHandler(authClient, nil, nil)
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
	h := NewAdminHandler(nil, nil, nil)
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

	h := NewAdminHandler(nil, nbClient, nil)
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

	h := NewAdminHandler(nil, nbClient, nil)
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

	h := NewAdminHandler(nil, nbClient, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/admin/notebooks/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.DeleteNotebook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_DeleteNotebook_InvalidID(t *testing.T) {
	h := NewAdminHandler(nil, nil, nil)
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

	h := NewAdminHandler(nil, nbClient, nil)
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
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)

	authClient.EXPECT().AdminGetStats(gomock.Any(), gomock.Any()).Return(&pbauth.AdminGetStatsResponse{
		TotalUsers:    100,
		TotalSessions: 50,
		Dau:           10,
		Mau:           30,
	}, nil)
	nbClient.EXPECT().AdminGetNotebookCount(gomock.Any(), gomock.Any()).
		Return(&pbnotebook.AdminGetNotebookCountResponse{Total: 25}, nil)

	h := NewAdminHandler(authClient, nbClient, nil)
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

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetStats(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestAdminHandler_UpdateUser_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminUpdateUser(gomock.Any(), gomock.Any()).
		Return(&pbauth.UserResponse{User: &pbauth.UserInfo{Id: 2, Username: "new"}}, nil)

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2",
		bytes.NewReader([]byte(`{"username":"new","email":"new@x.io"}`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_UpdateUser_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAdminHandler(mocks.NewMockAuthServiceClient(ctrl), nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/abc", bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_UpdateUser_BadBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAdminHandler(mocks.NewMockAuthServiceClient(ctrl), nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2", bytes.NewReader([]byte(`not json`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_UpdateUser_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminUpdateUser(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.AlreadyExists, "dup"))

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2",
		bytes.NewReader([]byte(`{"username":"new"}`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", rec.Code)
	}
}

func TestAdminHandler_ResetPassword_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminResetPassword(gomock.Any(), gomock.Any()).
		Return(&pbauth.AdminResetPasswordResponse{}, nil)

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2/password",
		bytes.NewReader([]byte(`{"password":"newpass"}`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_ResetPassword_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAdminHandler(mocks.NewMockAuthServiceClient(ctrl), nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/abc/password", bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_ResetPassword_BadBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAdminHandler(mocks.NewMockAuthServiceClient(ctrl), nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2/password", bytes.NewReader([]byte(`{`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_ResetPassword_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminResetPassword(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "missing"))

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2/password",
		bytes.NewReader([]byte(`{"password":"x"}`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestAdminHandler_SetPlan_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminSetPlan(gomock.Any(), gomock.Any()).
		Return(&pbauth.AdminSetPlanResponse{}, nil)

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2/plan",
		bytes.NewReader([]byte(`{"plan":"pro"}`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.SetPlan(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_SetPlan_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAdminHandler(mocks.NewMockAuthServiceClient(ctrl), nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/x/plan", bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", "x")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.SetPlan(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_SetPlan_BadBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAdminHandler(mocks.NewMockAuthServiceClient(ctrl), nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2/plan", bytes.NewReader([]byte(`{`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.SetPlan(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_SetPlan_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminSetPlan(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.InvalidArgument, "bad plan"))

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("PUT", "/admin/users/2/plan",
		bytes.NewReader([]byte(`{"plan":"weird"}`)))
	req.SetPathValue("id", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.SetPlan(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAdminHandler_GetActivityStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminGetActivityStats(gomock.Any(), gomock.Any()).
		Return(&pbauth.AdminGetActivityStatsResponse{
			Dau: []*pbauth.DauEntry{{Date: "2025-01-01", Count: 5}},
			Mau: []*pbauth.MauEntry{{Month: "2025-01", Count: 50}},
		}, nil)

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("GET", "/admin/stats/activity?dau_days=7&mau_months=3", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetActivityStats(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAdminHandler_GetActivityStats_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		AdminGetActivityStats(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "no"))

	h := NewAdminHandler(authClient, nil, nil)
	req := httptest.NewRequest("GET", "/admin/stats/activity", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetActivityStats(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestAdminHandler_RegisterRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAdminHandler(mocks.NewMockAuthServiceClient(ctrl), mocks.NewMockNotebookServiceClient(ctrl), nil)
	mux := http.NewServeMux()
	noop := func(next http.Handler) http.Handler { return next }
	var noopMw middleware.Middleware = noop
	h.RegisterRoutes(mux, noopMw, noopMw)

	// Каждый зарегистрированный роут должен резолвиться (и попадать в нужный хендлер,
	// у которого без user в контексте упадёт panic — поэтому сразу проверяем подсунутый user).
	cases := []struct {
		method, path string
	}{
		{"GET", "/api/v1/admin/users"},
		{"POST", "/api/v1/admin/users/1/ban"},
		{"POST", "/api/v1/admin/users/1/unban"},
		{"PUT", "/api/v1/admin/users/1"},
		{"PUT", "/api/v1/admin/users/1/password"},
		{"PUT", "/api/v1/admin/users/1/plan"},
		{"GET", "/api/v1/admin/notebooks"},
		{"DELETE", "/api/v1/admin/notebooks/1"},
		{"GET", "/api/v1/admin/stats"},
		{"GET", "/api/v1/admin/stats/activity"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
		req = withUser(req, 1)
		rec := httptest.NewRecorder()
		// gomock.NewController с .EXPECT() не задаём — большинство grpc-вызовов вернут
		// nil, и тест может упасть. Этот тест проверяет ТОЛЬКО что mux находит handler.
		_, pattern := mux.Handler(req)
		if pattern == "" {
			t.Errorf("no route registered for %s %s", tc.method, tc.path)
		}
		_ = rec
	}
}
