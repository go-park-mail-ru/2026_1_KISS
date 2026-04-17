package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

func setupServer(t *testing.T) (pb.RunnerServiceClient, *mocks.MockRunnerService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	runnerSvc := mocks.NewMockRunnerService(ctrl)
	srv := NewServer(runnerSvc)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterRunnerServiceServer(grpcServer, srv)

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
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return pb.NewRunnerServiceClient(conn), runnerSvc
}

func TestStopSession_Success(t *testing.T) {
	client, svc := setupServer(t)

	svc.EXPECT().StopSession(gomock.Any(), int64(1)).Return(nil)

	_, err := client.StopSession(context.Background(), &pb.StopSessionRequest{NotebookId: 1})
	if err != nil {
		t.Fatalf("stop session: %v", err)
	}
}

func TestExecuteBlock_Success(t *testing.T) {
	client, svc := setupServer(t)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(nil)
	svc.EXPECT().ExecuteBlock(gomock.Any(), int64(1), 0).Return(&domain.BlockExecutionResult{
		BlockID:    10,
		Position:   0,
		Stdout:     []string{"hello"},
		ExecutedAt: time.Now(),
		Duration:   100 * time.Millisecond,
	}, nil)

	resp, err := client.ExecuteBlock(context.Background(), &pb.ExecuteBlockRequest{
		NotebookId:    1,
		BlockPosition: 0,
	})
	if err != nil {
		t.Fatalf("execute block: %v", err)
	}
	if resp.GetResult().GetBlockId() != 10 {
		t.Errorf("want block id 10, got %d", resp.GetResult().GetBlockId())
	}
	if len(resp.GetResult().GetStdout()) != 1 || resp.GetResult().GetStdout()[0] != "hello" {
		t.Errorf("want stdout [hello], got %v", resp.GetResult().GetStdout())
	}
}

func TestExecuteFromPosition_Success(t *testing.T) {
	client, svc := setupServer(t)

	svc.EXPECT().StartSession(gomock.Any(), int64(1)).Return(nil)
	svc.EXPECT().ExecuteFromPosition(gomock.Any(), int64(1), 0).Return([]*domain.BlockExecutionResult{
		{BlockID: 10, Position: 0, ExecutedAt: time.Now(), Duration: 50 * time.Millisecond},
		{BlockID: 11, Position: 1, ExecutedAt: time.Now(), Duration: 60 * time.Millisecond},
	}, nil)

	resp, err := client.ExecuteFromPosition(context.Background(), &pb.ExecuteFromPositionRequest{
		NotebookId:    1,
		BlockPosition: 0,
	})
	if err != nil {
		t.Fatalf("execute from position: %v", err)
	}
	if len(resp.GetResults()) != 2 {
		t.Errorf("want 2 results, got %d", len(resp.GetResults()))
	}
}
