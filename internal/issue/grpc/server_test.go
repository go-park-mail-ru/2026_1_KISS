package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/issue/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/issue"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type testEnv struct {
	client    pb.IssueServiceClient
	issueRepo *mocks.MockIssueRepository
	conn      *grpc.ClientConn
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	issueRepo := mocks.NewMockIssueRepository(ctrl)
	issueUC := usecase.NewIssueService(issueRepo)
	srv := NewServer(issueUC)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterIssueServiceServer(grpcServer, srv)

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
		client:    pb.NewIssueServiceClient(conn),
		issueRepo: issueRepo,
		conn:      conn,
	}
}

func TestIssueService_GetByID_Success(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.Issue{
		ID:        1,
		Category:  domain.CategoryBug,
		Status:    domain.IssueStatusOpen,
		Content:   "Test issue content",
		UserID:    100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil)

	resp, err := env.client.GetByID(context.Background(), &pb.GetIssueRequest{
		Id: 1,
	})
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if resp.GetIssue().GetId() != 1 {
		t.Errorf("want issue id 1, got %d", resp.GetIssue().GetId())
	}
	if resp.GetIssue().GetCategory() != string(domain.CategoryBug) {
		t.Errorf("want category %s, got %s", domain.CategoryBug, resp.GetIssue().GetCategory())
	}
}

func TestIssueService_GetByID_NotFound(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetByID(gomock.Any(), int64(999)).Return(nil, domain.ErrNotFound)

	_, err := env.client.GetByID(context.Background(), &pb.GetIssueRequest{
		Id: 999,
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("want NotFound, got %v", st.Code())
	}
}

func TestIssueService_Create_Success(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, issue *domain.Issue) (int64, error) {
			issue.ID = 1
			issue.CreatedAt = time.Now()
			issue.UpdatedAt = time.Now()
			return 1, nil
		},
	)

	resp, err := env.client.Create(context.Background(), &pb.CreateIssueRequest{
		Category: string(domain.CategoryBug),
		Content:  "New issue",
		UserId:   100,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.GetIssue().GetId() != 1 {
		t.Errorf("want issue id 1, got %d", resp.GetIssue().GetId())
	}
	if resp.GetIssue().GetContent() != "New issue" {
		t.Errorf("want content 'New issue', got %s", resp.GetIssue().GetContent())
	}
}

func TestIssueService_Create_InvalidCategory(t *testing.T) {
	env := setup(t)

	_, err := env.client.Create(context.Background(), &pb.CreateIssueRequest{
		Category: "INVALID",
		Content:  "Test",
		UserId:   100,
	})
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", st.Code())
	}
}

func TestIssueService_Create_EmptyContent(t *testing.T) {
	env := setup(t)

	_, err := env.client.Create(context.Background(), &pb.CreateIssueRequest{
		Category: string(domain.CategoryBug),
		Content:  "",
		UserId:   100,
	})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", st.Code())
	}
}

func TestIssueService_GetAll_Success(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetAll(gomock.Any(), 20, 0, gomock.Any()).Return([]domain.Issue{
		{ID: 1, Category: domain.CategoryBug, Status: domain.IssueStatusOpen, Content: "Issue 1", UserID: 100},
		{ID: 2, Category: domain.CategoryIdea, Status: domain.IssueStatusInWork, Content: "Issue 2", UserID: 100},
	}, nil)

	resp, err := env.client.GetAll(context.Background(), &pb.GetAllIssuesRequest{
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(resp.GetIssues()) != 2 {
		t.Errorf("want 2 issues, got %d", len(resp.GetIssues()))
	}
}

func TestIssueService_GetAll_WithFilter(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetAll(gomock.Any(), 10, 0, gomock.Any()).DoAndReturn(
		func(_ context.Context, limit, offset int, filter *domain.IssueFilter) ([]domain.Issue, error) {
			if filter.Category != domain.CategoryBug {
				t.Errorf("want category %s, got %s", domain.CategoryBug, filter.Category)
			}
			if filter.Status != domain.IssueStatusOpen {
				t.Errorf("want status %s, got %s", domain.IssueStatusOpen, filter.Status)
			}
			return []domain.Issue{
				{ID: 1, Category: domain.CategoryBug, Status: domain.IssueStatusOpen, Content: "Bug", UserID: 100},
			}, nil
		},
	)

	resp, err := env.client.GetAll(context.Background(), &pb.GetAllIssuesRequest{
		Limit:    10,
		Offset:   0,
		Category: string(domain.CategoryBug),
		Status:   string(domain.IssueStatusOpen),
		UserId:   100,
	})
	if err != nil {
		t.Fatalf("GetAll with filter failed: %v", err)
	}
	if len(resp.GetIssues()) != 1 {
		t.Errorf("want 1 issue, got %d", len(resp.GetIssues()))
	}
}

func TestIssueService_Delete_Success(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().Delete(gomock.Any(), int64(1)).Return(nil)

	_, err := env.client.Delete(context.Background(), &pb.DeleteIssueRequest{
		Id: 1,
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestIssueService_Delete_NotFound(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().Delete(gomock.Any(), int64(999)).Return(domain.ErrNotFound)

	_, err := env.client.Delete(context.Background(), &pb.DeleteIssueRequest{
		Id: 999,
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("want NotFound, got %v", st.Code())
	}
}

func TestIssueService_AdminGetAllIssues_Success(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetAll(gomock.Any(), 20, 0, gomock.Any()).Return([]domain.Issue{
		{ID: 1, Category: domain.CategoryBug, Status: domain.IssueStatusOpen, Content: "Issue 1", UserID: 100},
		{ID: 2, Category: domain.CategoryProblem, Status: domain.IssueStatusClosed, Content: "Issue 2", UserID: 200},
		{ID: 3, Category: domain.CategoryFeedback, Status: domain.IssueStatusInWork, Content: "Issue 3", UserID: 300},
	}, nil)

	resp, err := env.client.AdminGetAllIssues(context.Background(), &pb.AdminGetAllIssuesRequest{
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("AdminGetAllIssues failed: %v", err)
	}
	if len(resp.GetIssues()) != 3 {
		t.Errorf("want 3 issues, got %d", len(resp.GetIssues()))
	}
}

func TestIssueService_AdminGetAllIssues_WithFilters(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetAll(gomock.Any(), 10, 0, gomock.Any()).DoAndReturn(
		func(_ context.Context, limit, offset int, filter *domain.IssueFilter) ([]domain.Issue, error) {
			if filter.Category != domain.CategoryBug {
				t.Errorf("want category %s, got %s", domain.CategoryBug, filter.Category)
			}
			return []domain.Issue{
				{ID: 1, Category: domain.CategoryBug, Status: domain.IssueStatusOpen, Content: "Bug", UserID: 100},
			}, nil
		},
	)

	resp, err := env.client.AdminGetAllIssues(context.Background(), &pb.AdminGetAllIssuesRequest{
		Limit:    10,
		Offset:   0,
		Category: string(domain.CategoryBug),
		Status:   string(domain.IssueStatusOpen),
	})
	if err != nil {
		t.Fatalf("AdminGetAllIssues with filter failed: %v", err)
	}
	if len(resp.GetIssues()) != 1 {
		t.Errorf("want 1 issue, got %d", len(resp.GetIssues()))
	}
}

func TestIssueService_AdminGetAllIssues_NormalizesLimit(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetAll(gomock.Any(), 20, 0, gomock.Any()).Return([]domain.Issue{}, nil)

	_, err := env.client.AdminGetAllIssues(context.Background(), &pb.AdminGetAllIssuesRequest{
		Limit:  0,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("AdminGetAllIssues failed: %v", err)
	}
}

func TestIssueService_AdminGetAllIssues_LimitTooHigh(t *testing.T) {
	env := setup(t)

	env.issueRepo.EXPECT().GetAll(gomock.Any(), 100, 0, gomock.Any()).Return([]domain.Issue{}, nil)

	_, err := env.client.AdminGetAllIssues(context.Background(), &pb.AdminGetAllIssuesRequest{
		Limit:  999,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("AdminGetAllIssues failed: %v", err)
	}
}
