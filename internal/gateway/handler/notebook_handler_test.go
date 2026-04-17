package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
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

	h := NewNotebookHandler(client)
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

	h := NewNotebookHandler(client)
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

	h := NewNotebookHandler(client)
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

	h := NewNotebookHandler(client)
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

	h := NewNotebookHandler(client)
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

	h := NewNotebookHandler(client)
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

	h := NewNotebookHandler(client)
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

	h := NewNotebookHandler(client)
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
	h := NewNotebookHandler(nil)
	req := httptest.NewRequest("GET", "/api/v1/notebooks", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}
