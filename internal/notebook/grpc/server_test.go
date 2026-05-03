package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

type testEnv struct {
	client       pb.NotebookServiceClient
	notebookRepo *mocks.MockNotebookRepository
	blockRepo    *mocks.MockBlockRepository
	permRepo     *mocks.MockPermissionRepository
	commentRepo  *mocks.MockCommentRepository
	conn         *grpc.ClientConn
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	ctrl := gomock.NewController(t)

	notebookRepo := mocks.NewMockNotebookRepository(ctrl)
	blockRepo := mocks.NewMockBlockRepository(ctrl)
	permRepo := mocks.NewMockPermissionRepository(ctrl)
	commentRepo := mocks.NewMockCommentRepository(ctrl)
	notebookUC := usecase.New(notebookRepo, blockRepo, permRepo, commentRepo)
	srv := NewServer(notebookUC, blockRepo)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterNotebookServiceServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("grpc serve error: %v", err)
		}
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		lis.Close()
	})

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return &testEnv{
		client:       pb.NewNotebookServiceClient(conn),
		notebookRepo: notebookRepo,
		blockRepo:    blockRepo,
		permRepo:     permRepo,
		commentRepo:  commentRepo,
		conn:         conn,
	}
}

func TestCreate_Success(t *testing.T) {
	env := setup(t)

	env.notebookRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, nb *domain.Notebook) (int64, error) {
			nb.ID = 1
			nb.CreatedAt = time.Now()
			return 1, nil
		},
	)

	resp, err := env.client.Create(context.Background(), &pb.CreateNotebookRequest{
		UserId: 1,
		Title:  "Test Notebook",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.GetNotebook().GetId() != 1 {
		t.Errorf("want notebook id 1, got %d", resp.GetNotebook().GetId())
	}
}

func TestGetByID_Success(t *testing.T) {
	env := setup(t)

	env.notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{
		ID:      1,
		OwnerID: 1,
		Title:   "Test",
	}, nil)
	env.blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return([]domain.Block{
		{ID: 10, NotebookID: 1, Type: "code", Content: "print('hello')"},
	}, nil)

	resp, err := env.client.GetByID(context.Background(), &pb.GetNotebookRequest{
		UserId:     1,
		NotebookId: 1,
	})
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if len(resp.GetNotebook().GetBlocks()) != 1 {
		t.Errorf("want 1 block, got %d", len(resp.GetNotebook().GetBlocks()))
	}
}

func TestGetByID_Forbidden(t *testing.T) {
	env := setup(t)

	env.notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{
		ID:      1,
		OwnerID: 99,
	}, nil)
	env.permRepo.EXPECT().GetPermission(gomock.Any(), int64(1), int64(1)).
		Return(nil, domain.ErrNotFound)

	_, err := env.client.GetByID(context.Background(), &pb.GetNotebookRequest{
		UserId:     1,
		NotebookId: 1,
	})
	if err == nil {
		t.Fatal("expected error for forbidden access")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.PermissionDenied {
		t.Errorf("want PermissionDenied, got %v", st.Code())
	}
}

func TestDelete_Success(t *testing.T) {
	env := setup(t)

	env.notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{
		ID:      1,
		OwnerID: 1,
	}, nil)
	env.notebookRepo.EXPECT().Delete(gomock.Any(), int64(1)).Return(nil)

	_, err := env.client.Delete(context.Background(), &pb.DeleteNotebookRequest{
		UserId:     1,
		NotebookId: 1,
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestAddBlock_Success(t *testing.T) {
	env := setup(t)

	env.notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{
		ID:      1,
		OwnerID: 1,
	}, nil)
	env.blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return([]domain.Block{}, nil)
	env.blockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, b *domain.Block) (int64, error) {
			b.ID = 10
			return 10, nil
		},
	)

	resp, err := env.client.AddBlock(context.Background(), &pb.AddBlockRequest{
		UserId:     1,
		NotebookId: 1,
		Type:       "code",
		Language:   "python",
		Content:    "print('hi')",
	})
	if err != nil {
		t.Fatalf("add block: %v", err)
	}
	if resp.GetBlock().GetId() != 10 {
		t.Errorf("want block id 10, got %d", resp.GetBlock().GetId())
	}
}

func TestGetBlocksByNotebookID_Success(t *testing.T) {
	env := setup(t)

	env.notebookRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Notebook{
		ID: 1, OwnerID: 1,
	}, nil)
	env.blockRepo.EXPECT().GetByNotebookID(gomock.Any(), int64(1)).Return([]domain.Block{
		{ID: 10, NotebookID: 1, Type: "code", Position: 0},
		{ID: 11, NotebookID: 1, Type: "text", Position: 1},
	}, nil).Times(2)

	resp, err := env.client.GetBlocksByNotebookID(context.Background(), &pb.GetBlocksRequest{
		NotebookId: 1,
		UserId:     1,
	})
	if err != nil {
		t.Fatalf("get blocks: %v", err)
	}
	if len(resp.GetBlocks()) != 2 {
		t.Errorf("want 2 blocks, got %d", len(resp.GetBlocks()))
	}
}

func TestListByUser_Success(t *testing.T) {
	env := setup(t)

	env.notebookRepo.EXPECT().GetByOwnerID(gomock.Any(), int64(1), 20, 0, "").Return([]domain.Notebook{
		{ID: 1, OwnerID: 1, Title: "NB1"},
	}, nil)
	env.notebookRepo.EXPECT().CountByOwnerID(gomock.Any(), int64(1), "").Return(1, nil)

	resp, err := env.client.ListByUser(context.Background(), &pb.ListNotebooksRequest{
		UserId: 1,
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.GetTotal() != 1 {
		t.Errorf("want total 1, got %d", resp.GetTotal())
	}
}
