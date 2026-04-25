package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

func TestNewWSHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	h := NewWSHandler(authClient, nbClient)
	if h == nil {
		t.Fatal("NewWSHandler returned nil")
	}
	if h.authClient != authClient || h.nbClient != nbClient {
		t.Errorf("clients not stored correctly")
	}
}

func TestWSHandler_RegisterRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), mocks.NewMockNotebookServiceClient(ctrl))
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Без сессии HandleNotebook вернёт 401, что подтверждает, что роут зарегистрирован
	// и принимает GET /api/v1/ws/notebooks/{id}.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/notebooks/42", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 (no cookie), got %d", rec.Code)
	}
}

func TestWSHandler_authenticate_NoCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), mocks.NewMockNotebookServiceClient(ctrl))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	uid, ok := h.authenticate(rec, req)
	if ok || uid != 0 {
		t.Errorf("want (0, false), got (%d, %v)", uid, ok)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestWSHandler_authenticate_InvalidSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		ValidateSession(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Unauthenticated, "invalid"))

	h := NewWSHandler(authClient, mocks.NewMockNotebookServiceClient(ctrl))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "broken"})
	rec := httptest.NewRecorder()
	uid, ok := h.authenticate(rec, req)
	if ok || uid != 0 {
		t.Errorf("want (0, false), got (%d, %v)", uid, ok)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestWSHandler_authenticate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		ValidateSession(gomock.Any(), gomock.Any()).
		Return(&pbauth.ValidateSessionResponse{User: &pbauth.UserInfo{Id: 7}}, nil)

	h := NewWSHandler(authClient, mocks.NewMockNotebookServiceClient(ctrl))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "ok"})
	rec := httptest.NewRecorder()
	uid, ok := h.authenticate(rec, req)
	if !ok || uid != 7 {
		t.Errorf("want (7, true), got (%d, %v)", uid, ok)
	}
}

func TestWSHandler_checkReadAccess_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		GetByID(gomock.Any(), gomock.Any()).
		Return(&pb.NotebookResponse{Notebook: &pb.NotebookInfo{Id: 1}}, nil)

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	rec := httptest.NewRecorder()
	if !h.checkReadAccess(rec, context.Background(), 1, 1) {
		t.Errorf("want true")
	}
}

func TestWSHandler_checkReadAccess_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		GetByID(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.NotFound, "not found"))

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	rec := httptest.NewRecorder()
	if h.checkReadAccess(rec, context.Background(), 1, 1) {
		t.Errorf("want false")
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestWSHandler_checkReadAccess_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		GetByID(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "forbidden"))

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	rec := httptest.NewRecorder()
	if h.checkReadAccess(rec, context.Background(), 1, 1) {
		t.Errorf("want false")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestWSHandler_checkReadAccess_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		GetByID(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "boom"))

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	rec := httptest.NewRecorder()
	if h.checkReadAccess(rec, context.Background(), 1, 1) {
		t.Errorf("want false")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestWSHandler_HandleNotebook_NoCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), mocks.NewMockNotebookServiceClient(ctrl))

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/notebooks/5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestWSHandler_HandleNotebook_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		ValidateSession(gomock.Any(), gomock.Any()).
		Return(&pbauth.ValidateSessionResponse{User: &pbauth.UserInfo{Id: 1}}, nil)

	h := NewWSHandler(authClient, mocks.NewMockNotebookServiceClient(ctrl))
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/notebooks/not-a-number", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "ok"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestWSHandler_HandleNotebook_AccessDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		ValidateSession(gomock.Any(), gomock.Any()).
		Return(&pbauth.ValidateSessionResponse{User: &pbauth.UserInfo{Id: 1}}, nil)

	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		GetByID(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "forbidden"))

	h := NewWSHandler(authClient, nbClient)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/notebooks/5", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "ok"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestWSHandler_HandleNotebook_AcceptFails(t *testing.T) {
	// Без заголовка Upgrade websocket.Accept падает, и мы выходим до
	// SubscribeNotebook — путь покрывает ветку err != nil после Accept.
	ctrl := gomock.NewController(t)
	authClient := mocks.NewMockAuthServiceClient(ctrl)
	authClient.EXPECT().
		ValidateSession(gomock.Any(), gomock.Any()).
		Return(&pbauth.ValidateSessionResponse{User: &pbauth.UserInfo{Id: 1}}, nil)

	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		GetByID(gomock.Any(), gomock.Any()).
		Return(&pb.NotebookResponse{Notebook: &pb.NotebookInfo{Id: 5}}, nil)

	h := NewWSHandler(authClient, nbClient)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/notebooks/5", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "ok"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// websocket.Accept пишет 4xx без заголовков апгрейда; нам важно,
	// что хендлер отработал без паники. Конкретный код зависит от версии
	// библиотеки, поэтому проверяем «не 5xx».
	if rec.Code >= 500 {
		t.Errorf("unexpected 5xx after failed accept: %d", rec.Code)
	}
}

func TestHandleMutation_UpdateBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		UpdateBlock(gomock.Any(), &pb.UpdateBlockRequest{
			UserId: 1, NotebookId: 2, BlockId: 3,
			Content: "hi", Language: "python",
		}).
		Return(&pb.BlockResponse{}, nil)

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	err := h.handleMutation(context.Background(), wsIncoming{
		Type: "update_block", BlockID: 3, Content: "hi", Language: "python",
	}, 1, 2)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
}

func TestHandleMutation_AddBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		AddBlock(gomock.Any(), &pb.AddBlockRequest{
			UserId: 1, NotebookId: 2, Type: "code", Language: "python",
		}).
		Return(&pb.BlockResponse{}, nil)

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	err := h.handleMutation(context.Background(), wsIncoming{
		Type: "add_block", BlockType: "code", Language: "python",
	}, 1, 2)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
}

func TestHandleMutation_DeleteBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		DeleteBlock(gomock.Any(), &pb.DeleteBlockRequest{
			UserId: 1, NotebookId: 2, BlockId: 9,
		}).
		Return(&pb.DeleteBlockResponse{}, nil)

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	err := h.handleMutation(context.Background(), wsIncoming{
		Type: "delete_block", BlockID: 9,
	}, 1, 2)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
}

func TestHandleMutation_UnknownType(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), mocks.NewMockNotebookServiceClient(ctrl))
	err := h.handleMutation(context.Background(), wsIncoming{Type: "weird"}, 1, 2)
	if err != nil {
		t.Errorf("want nil for unknown type, got %v", err)
	}
}

func TestHandleMutation_PropagatesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	nbClient := mocks.NewMockNotebookServiceClient(ctrl)
	nbClient.EXPECT().
		UpdateBlock(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "no"))

	h := NewWSHandler(mocks.NewMockAuthServiceClient(ctrl), nbClient)
	err := h.handleMutation(context.Background(), wsIncoming{Type: "update_block", BlockID: 1}, 1, 2)
	if err == nil {
		t.Errorf("want error from grpc, got nil")
	}
}

func TestEventToWS_BlockAdded(t *testing.T) {
	out := eventToWS(&pb.NotebookEvent{
		Type:    pb.NotebookEvent_BLOCK_ADDED,
		ActorId: 11,
		Payload: &pb.NotebookEvent_Block{Block: &pb.BlockInfo{Id: 2, Content: "x"}},
	})
	if out.Type != "block_added" || out.ActorID != 11 {
		t.Errorf("unexpected: %+v", out)
	}
	if len(out.Block) == 0 {
		t.Errorf("block payload missing")
	}
}

func TestEventToWS_BlockUpdated(t *testing.T) {
	out := eventToWS(&pb.NotebookEvent{
		Type:    pb.NotebookEvent_BLOCK_UPDATED,
		ActorId: 1,
		Payload: &pb.NotebookEvent_Block{Block: &pb.BlockInfo{Id: 5}},
	})
	if out.Type != "block_updated" {
		t.Errorf("want block_updated, got %s", out.Type)
	}
	if len(out.Block) == 0 {
		t.Errorf("block payload missing")
	}
}

func TestEventToWS_BlockDeleted(t *testing.T) {
	out := eventToWS(&pb.NotebookEvent{
		Type:    pb.NotebookEvent_BLOCK_DELETED,
		ActorId: 1,
		Payload: &pb.NotebookEvent_DeletedBlockId{DeletedBlockId: 99},
	})
	if out.Type != "block_deleted" || out.BlockID != 99 {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestEventToWS_NotebookUpdated(t *testing.T) {
	out := eventToWS(&pb.NotebookEvent{
		Type:    pb.NotebookEvent_NOTEBOOK_UPDATED,
		ActorId: 1,
	})
	if out.Type != "notebook_updated" {
		t.Errorf("want notebook_updated, got %s", out.Type)
	}
}

func TestErrorEvent_Mapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"forbidden", status.Error(codes.PermissionDenied, "no"), "forbidden"},
		{"not found", status.Error(codes.NotFound, "no"), "not found"},
		{"invalid input", status.Error(codes.InvalidArgument, "no"), "invalid input"},
		{"internal", status.Error(codes.Internal, "boom"), "internal error"},
		{"unknown plain error", errors.New("x"), "internal error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := errorEvent(tc.err)
			if out.Type != "error" || out.Message != tc.want {
				t.Errorf("got %+v, want type=error message=%s", out, tc.want)
			}
		})
	}
}

func TestMarshalBlock_Nil(t *testing.T) {
	if marshalBlock(nil) != nil {
		t.Errorf("want nil for nil input")
	}
}

func TestMarshalBlock_Fields(t *testing.T) {
	raw := marshalBlock(&pb.BlockInfo{
		Id:         1,
		NotebookId: 2,
		Type:       "code",
		Language:   "python",
		Content:    "print('hi')",
		Position:   3,
		CreatedAt:  100,
		UpdatedAt:  200,
	})
	if raw == nil {
		t.Fatal("want bytes, got nil")
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	for _, k := range []string{"id", "notebook_id", "type", "language", "content", "position", "created_at", "updated_at"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing key %q in marshalled block: %s", k, string(raw))
		}
	}
	if got["type"] != "code" {
		t.Errorf("want type=code, got %v", got["type"])
	}
	// content должен быть строкой как есть, не подвергаясь HTML-escape
	if !strings.Contains(string(raw), "print('hi')") {
		t.Errorf("content lost in marshalling: %s", string(raw))
	}
}
