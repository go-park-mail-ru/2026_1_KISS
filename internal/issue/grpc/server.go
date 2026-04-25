package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/issue/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/issue"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedIssueServiceServer
	issueUC usecase.IssueService
}

func NewServer(issueUC usecase.IssueService) *Server {
	return &Server{issueUC: issueUC}
}

func (s *Server) GetByID(ctx context.Context, req *pb.GetIssueRequest) (*pb.IssueResponse, error) {
	issue, err := s.issueUC.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.IssueResponse{Issue: issueToProto(issue)}, nil
}

func (s *Server) GetAll(ctx context.Context, req *pb.GetAllIssuesRequest) (*pb.GetAllIssuesResponse, error) {
	filter := &domain.IssueFilter{
		Category: domain.IssueCategory(req.GetCategory()),
		Status:   domain.IssueStatus(req.GetStatus()),
		UserID:   req.GetUserId(),
	}

	issues, err := s.issueUC.GetAll(ctx, int(req.GetLimit()), int(req.GetOffset()), filter)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	items := make([]*pb.IssueInfo, len(issues))
	for i := range issues {
		items[i] = issueToProto(&issues[i])
	}

	total := len(items)
	if total > int(^uint32(0)>>1) { // max int32
		return nil, status.Error(codes.Internal, "total count exceeds int32 limit")
	}

	return &pb.GetAllIssuesResponse{
		Issues: items,
		Total:  int32(total),
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	}, nil
}

func (s *Server) Create(ctx context.Context, req *pb.CreateIssueRequest) (*pb.IssueResponse, error) {
	issue := &domain.Issue{
		Category: domain.IssueCategory(req.GetCategory()),
		Content:  req.GetContent(),
		UserID:   req.GetUserId(),
	}

	id, err := s.issueUC.Create(ctx, issue)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	issue.ID = id

	logger.Info(ctx, "grpc.issue.Create", "issue_id", id, "user_id", issue.UserID)
	return &pb.IssueResponse{Issue: issueToProto(issue)}, nil
}

func (s *Server) Update(ctx context.Context, req *pb.UpdateIssueRequest) (*pb.IssueResponse, error) {
	// Для обновления нужен user_id, но его нет в UpdateIssueRequest
	// Придется получать из контекста или добавить в proto
	// Пока используем заглушку - нужно получить user_id из контекста аутентификации
	userID := getUserIDFromContext(ctx) // Реализуйте получение user_id из контекста

	issue := &domain.Issue{
		ID:       req.GetId(),
		Category: domain.IssueCategory(req.GetCategory()),
		Status:   domain.IssueStatus(req.GetStatus()),
		Content:  req.GetContent(),
		UserID:   userID,
	}

	if err := s.issueUC.Update(ctx, issue); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	logger.Info(ctx, "grpc.issue.Update", "issue_id", issue.ID)
	return &pb.IssueResponse{Issue: issueToProto(issue)}, nil
}

func (s *Server) Delete(ctx context.Context, req *pb.DeleteIssueRequest) (*pb.DeleteIssueResponse, error) {
	if err := s.issueUC.Delete(ctx, req.GetId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	logger.Info(ctx, "grpc.issue.Delete", "issue_id", req.GetId())
	return &pb.DeleteIssueResponse{}, nil
}

func (s *Server) AdminGetAllIssues(ctx context.Context, req *pb.AdminGetAllIssuesRequest) (*pb.AdminGetAllIssuesResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := int(req.GetOffset())
	if offset < 0 {
		offset = 0
	}

	filter := &domain.IssueFilter{
		Category: domain.IssueCategory(req.GetCategory()),
		Status:   domain.IssueStatus(req.GetStatus()),
	}

	issues, err := s.issueUC.GetAll(ctx, limit, offset, filter)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	items := make([]*pb.IssueInfo, len(issues))
	for i := range issues {
		items[i] = issueToProto(&issues[i])
	}

	total := len(items)
	if total > int(^uint32(0)>>1) { // max int32
		return nil, status.Error(codes.Internal, "total count exceeds int32 limit")
	}

	return &pb.AdminGetAllIssuesResponse{
		Issues: items,
		Total:  int32(total),
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	}, nil
}

// Вспомогательные функции

func issueToProto(issue *domain.Issue) *pb.IssueInfo {
	if issue == nil {
		return nil
	}
	return &pb.IssueInfo{
		Id:       issue.ID,
		Category: string(issue.Category),
		Status:   string(issue.Status),
		Content:  issue.Content,
		UserId:   issue.UserID,
	}
}

// Временная функция - замените на реальное получение user_id из контекста
func getUserIDFromContext(ctx context.Context) int64 {
	// Реализуйте получение user_id из контекста
	// Например, из метаданных gRPC или JWT токена
	return 0
}
