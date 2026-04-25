package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/issue"
)

func TestIssueHandler_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue: &pb.IssueInfo{Id: 1, UserId: 1, Category: "bug", Status: "open", Content: "test"},
	}, nil)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(createIssueRequest{Category: "bug", Content: "test bug"})
	req := httptest.NewRequest("POST", "/api/v1/issues", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestIssueHandler_Create_NoAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(createIssueRequest{Category: "bug", Content: "test"})
	req := httptest.NewRequest("POST", "/api/v1/issues", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestIssueHandler_Create_MissingCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(createIssueRequest{Content: "test"})
	req := httptest.NewRequest("POST", "/api/v1/issues", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_Create_MissingContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(createIssueRequest{Category: "bug"})
	req := httptest.NewRequest("POST", "/api/v1/issues", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_Create_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.InvalidArgument, "bad category"))

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(createIssueRequest{Category: "wrong", Content: "test"})
	req := httptest.NewRequest("POST", "/api/v1/issues", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_GetAll_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetAll(gomock.Any(), gomock.Any()).Return(&pb.GetAllIssuesResponse{
		Issues: []*pb.IssueInfo{
			{Id: 1, UserId: 1, Category: "bug", Status: "open", Content: "test"},
		},
		Total: 1, Limit: 20, Offset: 0,
	}, nil)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/issues", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestIssueHandler_GetAll_NoAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/issues", nil)
	rec := httptest.NewRecorder()

	h.GetAll(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestIssueHandler_GetByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue:    &pb.IssueInfo{Id: 1, UserId: 1, Category: "bug", Status: "open", Content: "test"},
		Messages: []*pb.IssueMessageInfo{},
	}, nil)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/issues/1", nil)
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestIssueHandler_GetByID_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue: &pb.IssueInfo{Id: 1, UserId: 2, Category: "bug", Status: "open", Content: "test"},
	}, nil)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/issues/1", nil)
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestIssueHandler_GetByID_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/issues/abc", nil)
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_GetByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/issues/999", nil)
	req.SetPathValue("id", "999")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestIssueHandler_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(&pb.DeleteIssueResponse{}, nil)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/issues/1", nil)
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestIssueHandler_Delete_NoAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/issues/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminGetStats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetStats(gomock.Any(), gomock.Any()).Return(&pb.IssueStatsResponse{
		Total: 10, Open: 5, InProgress: 3, Closed: 2,
		Bug: 4, Idea: 3, Problem: 2, Feedback: 1,
	}, nil)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/issues/stats", nil)
	rec := httptest.NewRecorder()

	h.AdminGetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminUpdateStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().AdminUpdateStatus(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue: &pb.IssueInfo{Id: 1, UserId: 1, Category: "bug", Status: "closed"},
	}, nil)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(patchIssueStatusRequest{Status: "closed"})
	req := httptest.NewRequest("PATCH", "/api/v1/admin/issues/1/status", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.AdminUpdateStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminUpdateStatus_MissingStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(patchIssueStatusRequest{})
	req := httptest.NewRequest("PATCH", "/api/v1/admin/issues/1/status", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.AdminUpdateStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminGetAll_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().AdminGetAllIssues(gomock.Any(), gomock.Any()).Return(&pb.AdminGetAllIssuesResponse{
		Issues: []*pb.IssueInfo{
			{Id: 1, UserId: 1, Category: "bug", Status: "open", Content: "test"},
		},
		Total: 1, Limit: 20, Offset: 0,
	}, nil)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/issues", nil)
	rec := httptest.NewRecorder()

	h.AdminGetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminAddResponse_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().AddMessage(gomock.Any(), gomock.Any()).Return(&pb.AddIssueMessageResponse{
		Message: &pb.IssueMessageInfo{Id: 1, IssueId: 1, UserId: 1, IsAdmin: true, Content: "reply"},
	}, nil)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(adminIssueResponseRequest{Content: "reply"})
	req := httptest.NewRequest("POST", "/api/v1/admin/issues/1/response", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.AdminAddResponse(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminAddResponse_NoAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(adminIssueResponseRequest{Content: "reply"})
	req := httptest.NewRequest("POST", "/api/v1/admin/issues/1/response", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.AdminAddResponse(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestIssueHandler_AddMessage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue: &pb.IssueInfo{Id: 1, UserId: 1, Category: "bug", Status: "open"},
	}, nil)
	client.EXPECT().AddMessage(gomock.Any(), gomock.Any()).Return(&pb.AddIssueMessageResponse{
		Message: &pb.IssueMessageInfo{Id: 1, IssueId: 1, UserId: 1, Content: "msg"},
	}, nil)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(addMessageRequest{Content: "msg"})
	req := httptest.NewRequest("POST", "/api/v1/issues/1/messages", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.AddMessage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestIssueHandler_AddMessage_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue: &pb.IssueInfo{Id: 1, UserId: 2, Category: "bug", Status: "open"},
	}, nil)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(addMessageRequest{Content: "msg"})
	req := httptest.NewRequest("POST", "/api/v1/issues/1/messages", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.AddMessage(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestIssueHandler_AddMessage_NoAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(addMessageRequest{Content: "msg"})
	req := httptest.NewRequest("POST", "/api/v1/issues/1/messages", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.AddMessage(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestIssueHandler_AddMessage_EmptyContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue: &pb.IssueInfo{Id: 1, UserId: 1, Category: "bug", Status: "open"},
	}, nil)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(addMessageRequest{})
	req := httptest.NewRequest("POST", "/api/v1/issues/1/messages", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.AddMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminGetByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&pb.IssueResponse{
		Issue:    &pb.IssueInfo{Id: 1, UserId: 1, Category: "bug", Status: "open", Content: "test"},
		Messages: []*pb.IssueMessageInfo{},
	}, nil)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/issues/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.AdminGetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminGetByID_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/issues/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.AdminGetByID(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminUpdateStatus_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(patchIssueStatusRequest{Status: "closed"})
	req := httptest.NewRequest("PATCH", "/api/v1/admin/issues/abc/status", bytes.NewReader(body))
	req.SetPathValue("id", "abc")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.AdminUpdateStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_Delete_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/issues/abc", nil)
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestIssueHandler_AdminAddResponse_EmptyContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockIssueServiceClient(ctrl)

	h := NewIssueHandler(client, nil)
	body, _ := json.Marshal(adminIssueResponseRequest{})
	req := httptest.NewRequest("POST", "/api/v1/admin/issues/1/response", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.AdminAddResponse(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}
