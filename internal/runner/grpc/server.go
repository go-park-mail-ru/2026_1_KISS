package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

type Server struct {
	pb.UnimplementedRunnerServiceServer
	runnerSvc runner_service.RunnerService
}

func NewServer(runnerSvc runner_service.RunnerService) *Server {
	return &Server{runnerSvc: runnerSvc}
}

func (s *Server) ExecuteFromPosition(ctx context.Context, req *pb.ExecuteFromPositionRequest) (*pb.ExecuteFromPositionResponse, error) {
	if err := s.runnerSvc.StartSession(ctx, req.GetNotebookId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	results, err := s.runnerSvc.ExecuteFromPosition(ctx, req.GetNotebookId(), int(req.GetBlockPosition()))
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	pbResults := make([]*pb.BlockExecutionResult, len(results))
	for i, r := range results {
		pbResults[i] = executionResultToProto(r)
	}
	return &pb.ExecuteFromPositionResponse{Results: pbResults}, nil
}

func (s *Server) ExecuteBlock(ctx context.Context, req *pb.ExecuteBlockRequest) (*pb.ExecuteBlockResponse, error) {
	if err := s.runnerSvc.StartSession(ctx, req.GetNotebookId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	result, err := s.runnerSvc.ExecuteBlock(ctx, req.GetNotebookId(), int(req.GetBlockPosition()))
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.ExecuteBlockResponse{Result: executionResultToProto(result)}, nil
}

func (s *Server) StopSession(ctx context.Context, req *pb.StopSessionRequest) (*pb.StopSessionResponse, error) {
	if err := s.runnerSvc.StopSession(ctx, req.GetNotebookId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.StopSessionResponse{}, nil
}

func executionResultToProto(r *domain.BlockExecutionResult) *pb.BlockExecutionResult {
	if r == nil {
		return nil
	}
	errMsg := ""
	if r.Error != nil {
		errMsg = r.Error.Error()
	}
	return &pb.BlockExecutionResult{
		BlockId:    r.BlockID,
		Position:   int32(r.Position), //nolint:gosec // block position fits int32
		Stdout:     r.Stdout,
		Stderr:     r.Stderr,
		Result:     r.Result,
		Error:      errMsg,
		ExecutedAt: r.ExecutedAt.Unix(),
		DurationNs: r.Duration.Nanoseconds(),
	}
}
