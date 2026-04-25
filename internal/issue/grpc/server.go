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
	msgUC   usecase.IssueMessageService
}

func NewServer(issueUC usecase.IssueService, msgUC usecase.IssueMessageService) *Server {
	return &Server{issueUC: issueUC, msgUC: msgUC}
}

func (s *Server) GetByID(ctx context.Context, req *pb.GetIssueRequest) (*pb.IssueResponse, error) {
	issue, err := s.issueUC.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	resp := &pb.IssueResponse{Issue: issueToProto(issue)}

	if s.msgUC != nil {
		msgs, err := s.msgUC.GetByIssueID(ctx, issue.ID)
		if err != nil {
			return nil, grpcutil.DomainToGRPCError(err)
		}
		resp.Messages = msgsToProto(msgs)
	}

	return resp, nil
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
	if total > int(^uint32(0)>>1) {
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
	issue := &domain.Issue{
		ID:       req.GetId(),
		Category: domain.IssueCategory(req.GetCategory()),
		Status:   domain.IssueStatus(req.GetStatus()),
		Content:  req.GetContent(),
		UserID:   req.GetUserId(),
	}

	if err := s.issueUC.Update(ctx, issue); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	logger.Info(ctx, "grpc.issue.Update", "issue_id", issue.ID)
	return &pb.IssueResponse{Issue: issueToProto(issue)}, nil
}

func (s *Server) Delete(ctx context.Context, req *pb.DeleteIssueRequest) (*pb.DeleteIssueResponse, error) {
	if err := s.issueUC.Delete(ctx, req.GetId(), req.GetUserId()); err != nil {
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
		UserID:   req.GetUserId(),
		Content:  req.GetContent(),
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
	if total > int(^uint32(0)>>1) {
		return nil, status.Error(codes.Internal, "total count exceeds int32 limit")
	}

	return &pb.AdminGetAllIssuesResponse{
		Issues: items,
		Total:  int32(total),
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	}, nil
}

func (s *Server) AdminUpdateStatus(ctx context.Context, req *pb.AdminUpdateIssueStatusRequest) (*pb.IssueResponse, error) {
	if err := s.issueUC.AdminUpdateStatus(ctx, req.GetId(), domain.IssueStatus(req.GetStatus())); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	issue, err := s.issueUC.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	logger.Info(ctx, "grpc.issue.AdminUpdateStatus", "issue_id", req.GetId(), "status", req.GetStatus())
	return &pb.IssueResponse{Issue: issueToProto(issue)}, nil
}

func (s *Server) AddMessage(ctx context.Context, req *pb.AddIssueMessageRequest) (*pb.AddIssueMessageResponse, error) {
	if s.msgUC == nil {
		return nil, status.Error(codes.Unimplemented, "messages not configured")
	}

	msg := &domain.IssueMessage{
		IssueID: req.GetIssueId(),
		UserID:  req.GetUserId(),
		IsAdmin: req.GetIsAdmin(),
		Content: req.GetContent(),
	}

	id, err := s.msgUC.AddMessage(ctx, msg)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	msg.ID = id

	logger.Info(ctx, "grpc.issue.AddMessage", "message_id", id, "issue_id", msg.IssueID)
	return &pb.AddIssueMessageResponse{Message: msgToProto(msg)}, nil
}

func (s *Server) GetStats(ctx context.Context, _ *pb.GetIssueStatsRequest) (*pb.IssueStatsResponse, error) {
	stats, err := s.issueUC.GetStats(ctx)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	return &pb.IssueStatsResponse{
		Total:      stats.Total,
		Open:       stats.Open,
		InProgress: stats.InProgress,
		Closed:     stats.Closed,
		Bug:        stats.Bug,
		Idea:       stats.Idea,
		Problem:    stats.Problem,
		Feedback:   stats.Feedback,
	}, nil
}

func issueToProto(issue *domain.Issue) *pb.IssueInfo {
	if issue == nil {
		return nil
	}
	return &pb.IssueInfo{
		Id:        issue.ID,
		Category:  string(issue.Category),
		Status:    string(issue.Status),
		Content:   issue.Content,
		UserId:    issue.UserID,
		CreatedAt: issue.CreatedAt.Unix(),
		UpdatedAt: issue.UpdatedAt.Unix(),
	}
}

func msgToProto(msg *domain.IssueMessage) *pb.IssueMessageInfo {
	if msg == nil {
		return nil
	}
	return &pb.IssueMessageInfo{
		Id:        msg.ID,
		IssueId:   msg.IssueID,
		UserId:    msg.UserID,
		IsAdmin:   msg.IsAdmin,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.Unix(),
	}
}

func msgsToProto(msgs []domain.IssueMessage) []*pb.IssueMessageInfo {
	result := make([]*pb.IssueMessageInfo, len(msgs))
	for i := range msgs {
		result[i] = msgToProto(&msgs[i])
	}
	return result
}
