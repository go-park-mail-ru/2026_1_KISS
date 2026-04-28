package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

func withUser(r *http.Request, userID int64) *http.Request {
	user := &domain.User{ID: userID}
	ctx := middleware.SetUserInContext(r.Context(), user)
	return r.WithContext(ctx)
}

func TestNotebookHandler_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&pb.NotebookResponse{
		Notebook: &pb.NotebookInfo{Id: 1, OwnerId: 1, Title: "Test"},
	}, nil)

	h := NewNotebookHandler(client, nil)
	body, _ := json.Marshal(createNotebookRequest{Title: "Test"})
	req := httptest.NewRequest("POST", "/api/v1/notebooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestNotebookHandler_List_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().ListByUser(gomock.Any(), gomock.Any()).Return(&pb.ListNotebooksResponse{
		Notebooks: []*pb.NotebookInfo{{Id: 1, Title: "NB1"}},
		Total:     1,
		Limit:     20,
	}, nil)

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks?limit=20", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_GetByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&pb.NotebookResponse{
		Notebook: &pb.NotebookInfo{Id: 1, Title: "Test", Blocks: []*pb.BlockInfo{
			{Id: 10, Type: "code", Position: 0},
		}},
	}, nil)

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(&pb.DeleteNotebookResponse{}, nil)

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1", nil)
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_Update_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&pb.NotebookResponse{
		Notebook: &pb.NotebookInfo{Id: 1, Title: "Updated"},
	}, nil)

	h := NewNotebookHandler(client, nil)
	body, _ := json.Marshal(updateNotebookRequest{Title: "Updated", IsPublic: true})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_AddBlock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().AddBlock(gomock.Any(), gomock.Any()).Return(&pb.BlockResponse{
		Block: &pb.BlockInfo{Id: 10, Type: "code", Position: 0},
	}, nil)

	h := NewNotebookHandler(client, nil)
	body, _ := json.Marshal(createBlockRequest{Type: "code", Language: "python", Content: "print('hi')"})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/blocks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.AddBlock(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

func TestNotebookHandler_UpdateBlock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().UpdateBlock(gomock.Any(), gomock.Any()).Return(&pb.BlockResponse{
		Block: &pb.BlockInfo{Id: 10, Content: "updated"},
	}, nil)

	h := NewNotebookHandler(client, nil)
	body, _ := json.Marshal(updateBlockRequest{Content: "updated"})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/blocks/10", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "10")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.UpdateBlock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_DeleteBlock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().DeleteBlock(gomock.Any(), gomock.Any()).Return(&pb.DeleteBlockResponse{}, nil)

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/blocks/10", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("blockID", "10")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.DeleteBlock(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_Unauthorized(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestNotebookHandler_ListShared_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().ListSharedWithUser(gomock.Any(), gomock.Any()).Return(&pb.ListNotebooksResponse{
		Notebooks: []*pb.NotebookInfo{{Id: 5, OwnerId: 2, Title: "Shared NB"}},
		Total:     1,
		Limit:     20,
	}, nil)

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks/shared?limit=20", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListShared(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_ListShared_Unauthorized(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks/shared", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	h.ListShared(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestNotebookHandler_ListShared_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().ListSharedWithUser(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "internal"))

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks/shared", nil)
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListShared(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestNotebookHandler_ListPermissions_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	client.EXPECT().ListPermissions(gomock.Any(), gomock.Any()).Return(&pb.ListPermissionsResponse{
		Permissions: []*pb.PermissionInfo{
			{NotebookId: 1, UserId: 2, PermissionLevel: "editor"},
		},
	}, nil)
	authClient.EXPECT().GetUserByID(gomock.Any(), &pbauth.GetUserByIDRequest{UserId: 2}).Return(
		&pbauth.UserResponse{User: &pbauth.UserInfo{Email: "user@example.com"}},
		nil,
	)

	h := NewNotebookHandler(client, authClient)
	req := httptest.NewRequest("GET", "/api/v1/notebooks/1/permissions", nil)
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListPermissions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_ListPermissions_InvalidID(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks/abc/permissions", nil)
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ListPermissions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_ListPermissions_Unauthorized(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks/1/permissions", nil)
	req.SetPathValue("id", "1")
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	h.ListPermissions(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermissionByIdentifier_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().GetUserByIdentifier(gomock.Any(), gomock.Any()).Return(&pbauth.UserResponse{
		User: &pbauth.UserInfo{Id: 2, Username: "target", Email: "target@test.com"},
	}, nil)
	client.EXPECT().GrantPermission(gomock.Any(), gomock.Any()).
		Return(&pb.GrantPermissionResponse{}, nil)

	h := NewNotebookHandler(client, authClient)
	body, _ := json.Marshal(grantPermissionByIdentifierRequest{Identifier: "target@test.com", Level: "editor"})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/permissions/grant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermissionByIdentifier(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermissionByIdentifier_EmptyIdentifier(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	body, _ := json.Marshal(grantPermissionByIdentifierRequest{Identifier: "", Level: "editor"})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/permissions/grant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermissionByIdentifier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermissionByIdentifier_EmptyLevel(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	body, _ := json.Marshal(grantPermissionByIdentifierRequest{Identifier: "user@test.com", Level: ""})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/permissions/grant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermissionByIdentifier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermissionByIdentifier_AuthError(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)

	authClient.EXPECT().GetUserByIdentifier(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewNotebookHandler(nil, authClient)
	body, _ := json.Marshal(grantPermissionByIdentifierRequest{Identifier: "unknown@test.com", Level: "editor"})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/permissions/grant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermissionByIdentifier(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermissionByIdentifier_InvalidID(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/notebooks/abc/permissions/grant", nil)
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermissionByIdentifier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermission_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().GrantPermission(gomock.Any(), gomock.Any()).
		Return(&pb.GrantPermissionResponse{}, nil)

	h := NewNotebookHandler(client, nil)
	body, _ := json.Marshal(grantPermissionRequest{Level: "editor"})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/permissions/2", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("userID", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermission(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermission_InvalidNotebookID(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/notebooks/abc/permissions/2", nil)
	req.SetPathValue("id", "abc")
	req.SetPathValue("userID", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermission(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermission_InvalidUserID(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/permissions/abc", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("userID", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermission(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_GrantPermission_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().GrantPermission(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "forbidden"))

	h := NewNotebookHandler(client, nil)
	body, _ := json.Marshal(grantPermissionRequest{Level: "editor"})
	req := httptest.NewRequest("POST", "/api/v1/notebooks/1/permissions/2", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req.SetPathValue("userID", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.GrantPermission(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestNotebookHandler_RevokePermission_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().RevokePermission(gomock.Any(), gomock.Any()).
		Return(&pb.RevokePermissionResponse{}, nil)

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/permissions/2", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("userID", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.RevokePermission(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d", rec.Code)
	}
}

func TestNotebookHandler_RevokePermission_InvalidNotebookID(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/abc/permissions/2", nil)
	req.SetPathValue("id", "abc")
	req.SetPathValue("userID", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.RevokePermission(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_RevokePermission_InvalidUserID(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/permissions/abc", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("userID", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.RevokePermission(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_ReorderBlocks_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().ReorderBlocks(gomock.Any(), gomock.Any()).
		Return(&pb.ReorderBlocksResponse{}, nil)

	h := NewNotebookHandler(client, nil)
	body, _ := json.Marshal(reorderBlocksRequest{BlockIDs: []int64{3, 1, 2}})
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/reorder", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ReorderBlocks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestNotebookHandler_ReorderBlocks_Unauthorized(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/1/reorder", nil)
	req.SetPathValue("id", "1")
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	h.ReorderBlocks(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestNotebookHandler_ReorderBlocks_InvalidID(t *testing.T) {
	h := NewNotebookHandler(nil, nil)
	req := httptest.NewRequest("PUT", "/api/v1/notebooks/abc/reorder", nil)
	req.SetPathValue("id", "abc")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.ReorderBlocks(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestNotebookHandler_RevokePermission_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockNotebookServiceClient(ctrl)

	client.EXPECT().RevokePermission(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewNotebookHandler(client, nil)
	req := httptest.NewRequest("DELETE", "/api/v1/notebooks/1/permissions/2", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("userID", "2")
	req = withUser(req, 1)
	rec := httptest.NewRecorder()

	h.RevokePermission(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}
