package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/ctxutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

type Server struct {
	pb.UnimplementedRunnerServiceServer
	runnerSvc runner_service.RunnerService
	nbClient  pbnotebook.NotebookServiceClient
}

func NewServer(runnerSvc runner_service.RunnerService, nbClient pbnotebook.NotebookServiceClient) *Server {
	return &Server{runnerSvc: runnerSvc, nbClient: nbClient}
}

func (s *Server) ExecuteFromPosition(ctx context.Context, req *pb.ExecuteFromPositionRequest) (*pb.ExecuteFromPositionResponse, error) {
	if err := s.checkNotebookAccess(ctx, req.GetUserId(), req.GetNotebookId()); err != nil {
		return nil, err
	}
	ctx = ctxutil.SetUserID(ctx, req.GetUserId())

	if _, err := s.runnerSvc.StartSession(ctx, req.GetNotebookId(), req.GetUserId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	results, err := s.runnerSvc.ExecuteFromPosition(ctx, req.GetNotebookId(), req.GetUserId(), int(req.GetBlockPosition()))
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	pbResults := make([]*pb.BlockExecutionResult, len(results))
	for i, r := range results {
		pbResults[i] = executionResultToProto(r)
	}

	s.saveBlockOutputs(ctx, results)

	return &pb.ExecuteFromPositionResponse{Results: pbResults}, nil
}

func (s *Server) ExecuteBlock(ctx context.Context, req *pb.ExecuteBlockRequest) (*pb.ExecuteBlockResponse, error) {
	if err := s.checkNotebookAccess(ctx, req.GetUserId(), req.GetNotebookId()); err != nil {
		return nil, err
	}
	ctx = ctxutil.SetUserID(ctx, req.GetUserId())

	if _, err := s.runnerSvc.StartSession(ctx, req.GetNotebookId(), req.GetUserId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	result, err := s.runnerSvc.ExecuteBlock(ctx, req.GetNotebookId(), req.GetUserId(), int(req.GetBlockPosition()))
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	s.saveBlockOutputs(ctx, []*domain.BlockExecutionResult{result})

	return &pb.ExecuteBlockResponse{Result: executionResultToProto(result)}, nil
}

func (s *Server) StopSession(ctx context.Context, req *pb.StopSessionRequest) (*pb.StopSessionResponse, error) {
	if err := s.checkNotebookAccess(ctx, req.GetUserId(), req.GetNotebookId()); err != nil {
		return nil, err
	}
	ctx = ctxutil.SetUserID(ctx, req.GetUserId())

	if err := s.runnerSvc.StopSession(ctx, req.GetNotebookId(), req.GetUserId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.StopSessionResponse{}, nil
}

func (s *Server) checkNotebookAccess(ctx context.Context, userID, notebookID int64) error {
	if userID == 0 {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	_, err := s.nbClient.GetByID(ctx, &pbnotebook.GetNotebookRequest{
		UserId:     userID,
		NotebookId: notebookID,
	})
	return err
}

func (s *Server) saveBlockOutputs(ctx context.Context, results []*domain.BlockExecutionResult) {
	for _, r := range results {
		if r == nil || r.Error != nil {
			continue
		}
		outputs := resultToOutputProtos(r)
		if len(outputs) == 0 {
			continue
		}
		_, err := s.nbClient.SaveBlockOutputs(ctx, &pbnotebook.SaveBlockOutputsRequest{
			BlockId: r.BlockID,
			Outputs: outputs,
		})
		if err != nil {
			logger.Error(ctx, "runner.SaveBlockOutputs", "error", err, "block_id", r.BlockID)
		}
	}
}

func resultToOutputProtos(r *domain.BlockExecutionResult) []*pbnotebook.BlockOutputInfo {
	var outputs []*pbnotebook.BlockOutputInfo
	pos := int32(0)

	if stdout := strings.Join(r.Stdout, "\n"); stdout != "" {
		outputs = append(outputs, &pbnotebook.BlockOutputInfo{
			Position: pos, OutputType: "stdout", Content: stdout,
		})
	}
	pos++

	if stderr := strings.Join(r.Stderr, "\n"); stderr != "" {
		outputs = append(outputs, &pbnotebook.BlockOutputInfo{
			Position: pos, OutputType: "stderr", Content: stderr,
		})
	}
	pos++

	if r.Result != "" {
		outputs = append(outputs, &pbnotebook.BlockOutputInfo{
			Position: pos, OutputType: "result", Content: r.Result,
		})
	}
	pos++

	for _, item := range r.Outputs {
		outputs = append(outputs, &pbnotebook.BlockOutputInfo{
			Position: pos, OutputType: item.MimeType, Content: item.Data,
		})
		pos++
	}

	return outputs
}

func (s *Server) GetSessionStats(ctx context.Context, req *pb.GetSessionStatsRequest) (*pb.GetSessionStatsResponse, error) {
	if err := s.checkNotebookAccess(ctx, req.GetUserId(), req.GetNotebookId()); err != nil {
		return nil, err
	}

	stats, err := s.runnerSvc.GetSessionStats(ctx, req.GetNotebookId(), req.GetUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	snapshotAge := int64(-1)
	if stats.SnapshotAge > 0 {
		snapshotAge = int64(stats.SnapshotAge.Seconds())
	}

	return &pb.GetSessionStatsResponse{
		CpuPercent:         stats.CPUPercent,
		MemoryUsage:        stats.MemoryUsage,
		MemoryLimit:        stats.MemoryLimit,
		MemoryPercent:      stats.MemoryPercent,
		CpuCores:           stats.CPUCores,
		DiskLimitBytes:     stats.DiskLimitBytes,
		GpuAvailable:       stats.GPUAvailable,
		QueuePosition:      stats.QueuePosition,
		SnapshotAgeSeconds: snapshotAge,
		SnapshotSizeBytes:  stats.SnapshotSizeBytes,
		SessionState:       stats.SessionState,
	}, nil
}

func (s *Server) ExecuteBlockStream(req *pb.ExecuteBlockRequest, stream pb.RunnerService_ExecuteBlockStreamServer) error {
	ctx := stream.Context()
	if err := s.checkNotebookAccess(ctx, req.GetUserId(), req.GetNotebookId()); err != nil {
		return err
	}
	ctx = ctxutil.SetUserID(ctx, req.GetUserId())

	if _, err := s.runnerSvc.StartSession(ctx, req.GetNotebookId(), req.GetUserId()); err != nil {
		return grpcutil.DomainToGRPCError(err)
	}

	result, err := s.runnerSvc.ExecuteBlockStreaming(ctx, req.GetNotebookId(), req.GetUserId(), int(req.GetBlockPosition()), func(chunkType, data string) {
		_ = stream.Send(&pb.ExecutionChunk{
			ChunkType: chunkType,
			Data:      data,
		})
	})
	if err != nil {
		return grpcutil.DomainToGRPCError(err)
	}

	s.saveBlockOutputs(ctx, []*domain.BlockExecutionResult{result})

	return stream.Send(&pb.ExecutionChunk{
		ChunkType:   "complete",
		FinalResult: executionResultToProto(result),
	})
}

func executionResultToProto(r *domain.BlockExecutionResult) *pb.BlockExecutionResult {
	if r == nil {
		return nil
	}
	errMsg := ""
	if r.Error != nil {
		errMsg = r.Error.Error()
	}
	pbOutputs := make([]*pb.OutputItem, len(r.Outputs))
	for i, o := range r.Outputs {
		pbOutputs[i] = &pb.OutputItem{MimeType: o.MimeType, Data: o.Data}
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
		Outputs:    pbOutputs,
	}
}
