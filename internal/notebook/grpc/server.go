package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

// EventHub is the subset of hub.Hub used by the gRPC server.
type EventHub interface {
	Subscribe(notebookID int64, connID string, bufSize int) chan *pb.NotebookEvent
	Unsubscribe(notebookID int64, connID string)
}

type noopHub struct{}

func (noopHub) Subscribe(_ int64, _ string, _ int) chan *pb.NotebookEvent {
	return make(chan *pb.NotebookEvent)
}
func (noopHub) Unsubscribe(_ int64, _ string) {}

type Server struct {
	pb.UnimplementedNotebookServiceServer
	notebookUC usecase.NotebookService
	blockRepo  repository.BlockRepository
	hub        EventHub
}

func NewServer(notebookUC usecase.NotebookService, blockRepo repository.BlockRepository, hubs ...EventHub) *Server {
	var hub EventHub = noopHub{}
	if len(hubs) > 0 && hubs[0] != nil {
		hub = hubs[0]
	}
	return &Server{notebookUC: notebookUC, blockRepo: blockRepo, hub: hub}
}

func (s *Server) Create(ctx context.Context, req *pb.CreateNotebookRequest) (*pb.NotebookResponse, error) {
	nb, err := s.notebookUC.Create(ctx, req.GetUserId(), req.GetTitle())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.NotebookResponse{Notebook: notebookToProto(nb)}, nil
}

func (s *Server) GetByID(ctx context.Context, req *pb.GetNotebookRequest) (*pb.NotebookResponse, error) {
	nb, err := s.notebookUC.GetByID(ctx, req.GetUserId(), req.GetNotebookId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.NotebookResponse{Notebook: notebookToProto(nb)}, nil
}

func (s *Server) ListByUser(ctx context.Context, req *pb.ListNotebooksRequest) (*pb.ListNotebooksResponse, error) {
	notebooks, total, err := s.notebookUC.ListByUser(ctx, req.GetUserId(), int(req.GetLimit()), int(req.GetOffset()), req.GetSearch())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	items := make([]*pb.NotebookInfo, len(notebooks))
	for i := range notebooks {
		items[i] = notebookToProto(&notebooks[i])
	}
	return &pb.ListNotebooksResponse{
		Notebooks: items,
		Total:     int32(total), //nolint:gosec // total notebooks count fits int32
		Limit:     req.GetLimit(),
		Offset:    req.GetOffset(),
	}, nil
}

func (s *Server) Update(ctx context.Context, req *pb.UpdateNotebookRequest) (*pb.NotebookResponse, error) {
	nb, err := s.notebookUC.Update(ctx, req.GetUserId(), req.GetNotebookId(), req.GetTitle(), req.GetIsPublic())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.NotebookResponse{Notebook: notebookToProto(nb)}, nil
}

func (s *Server) Delete(ctx context.Context, req *pb.DeleteNotebookRequest) (*pb.DeleteNotebookResponse, error) {
	if err := s.notebookUC.Delete(ctx, req.GetUserId(), req.GetNotebookId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.DeleteNotebookResponse{}, nil
}

func (s *Server) AddBlock(ctx context.Context, req *pb.AddBlockRequest) (*pb.BlockResponse, error) {
	block := &domain.Block{
		Type:     req.GetType(),
		Language: req.GetLanguage(),
		Content:  req.GetContent(),
	}
	b, err := s.notebookUC.AddBlock(ctx, req.GetUserId(), req.GetNotebookId(), block)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.BlockResponse{Block: blockToProto(b)}, nil
}

func (s *Server) UpdateBlock(ctx context.Context, req *pb.UpdateBlockRequest) (*pb.BlockResponse, error) {
	b, err := s.notebookUC.UpdateBlock(ctx, req.GetUserId(), req.GetNotebookId(), req.GetBlockId(), req.GetContent(), req.GetType(), req.GetLanguage())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.BlockResponse{Block: blockToProto(b)}, nil
}

func (s *Server) DeleteBlock(ctx context.Context, req *pb.DeleteBlockRequest) (*pb.DeleteBlockResponse, error) {
	if err := s.notebookUC.DeleteBlock(ctx, req.GetUserId(), req.GetNotebookId(), req.GetBlockId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.DeleteBlockResponse{}, nil
}

func (s *Server) GetBlocksByNotebookID(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlocksResponse, error) {
	if _, err := s.notebookUC.GetByID(ctx, req.GetUserId(), req.GetNotebookId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	blocks, err := s.blockRepo.GetByNotebookID(ctx, req.GetNotebookId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	items := make([]*pb.BlockInfo, len(blocks))
	for i := range blocks {
		items[i] = blockToProto(&blocks[i])
	}
	return &pb.GetBlocksResponse{Blocks: items}, nil
}

func (s *Server) AdminListNotebooks(ctx context.Context, req *pb.AdminListNotebooksRequest) (*pb.AdminListNotebooksResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	notebooks, err := s.notebookUC.ListAll(ctx, limit, int(req.GetOffset()), req.GetSearch())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	total, err := s.notebookUC.CountAll(ctx, req.GetSearch())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	items := make([]*pb.NotebookInfo, len(notebooks))
	for i := range notebooks {
		items[i] = notebookToProto(&notebooks[i])
	}
	return &pb.AdminListNotebooksResponse{
		Notebooks: items,
		Total:     int32(total), //nolint:gosec // total notebooks count fits int32
	}, nil
}

func (s *Server) AdminDeleteNotebook(ctx context.Context, req *pb.AdminDeleteNotebookRequest) (*pb.DeleteNotebookResponse, error) {
	if err := s.notebookUC.AdminDelete(ctx, req.GetNotebookId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.DeleteNotebookResponse{}, nil
}

func (s *Server) AdminSetUserNotebooksPrivate(ctx context.Context, req *pb.AdminSetUserNotebooksPrivateRequest) (*pb.AdminSetUserNotebooksPrivateResponse, error) {
	if err := s.notebookUC.SetAllPrivateByOwner(ctx, req.GetOwnerId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.AdminSetUserNotebooksPrivateResponse{}, nil
}

func (s *Server) AdminGetNotebookCount(ctx context.Context, _ *pb.AdminGetNotebookCountRequest) (*pb.AdminGetNotebookCountResponse, error) {
	count, err := s.notebookUC.CountAll(ctx, "")
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.AdminGetNotebookCountResponse{Total: int64(count)}, nil
}

func (s *Server) GrantPermission(ctx context.Context, req *pb.GrantPermissionRequest) (*pb.GrantPermissionResponse, error) {
	err := s.notebookUC.GrantPermission(ctx, req.GetRequesterId(), req.GetNotebookId(), req.GetTargetUserId(), req.GetLevel())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.GrantPermissionResponse{}, nil
}

func (s *Server) RevokePermission(ctx context.Context, req *pb.RevokePermissionRequest) (*pb.RevokePermissionResponse, error) {
	err := s.notebookUC.RevokePermission(ctx, req.GetRequesterId(), req.GetNotebookId(), req.GetTargetUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.RevokePermissionResponse{}, nil
}

func (s *Server) ListPermissions(ctx context.Context, req *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	perms, err := s.notebookUC.ListPermissions(ctx, req.GetRequesterId(), req.GetNotebookId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	items := make([]*pb.PermissionInfo, len(perms))
	for i, p := range perms {
		items[i] = &pb.PermissionInfo{
			NotebookId:      p.NotebookID,
			UserId:          p.UserID,
			PermissionLevel: p.PermissionLevel,
		}
	}
	return &pb.ListPermissionsResponse{Permissions: items}, nil
}

func (s *Server) ListSharedWithUser(ctx context.Context, req *pb.ListSharedWithUserRequest) (*pb.ListNotebooksResponse, error) {
	notebooks, total, err := s.notebookUC.ListSharedWithUser(ctx, req.GetUserId(), int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	items := make([]*pb.NotebookInfo, len(notebooks))
	for i := range notebooks {
		items[i] = notebookToProto(&notebooks[i])
	}
	return &pb.ListNotebooksResponse{
		Notebooks: items,
		Total:     int32(total), //nolint:gosec // total count fits int32
		Limit:     req.GetLimit(),
		Offset:    req.GetOffset(),
	}, nil
}

func (s *Server) SubscribeNotebook(req *pb.SubscribeNotebookRequest, stream pb.NotebookService_SubscribeNotebookServer) error {
	notebookID := req.GetNotebookId()
	userID := req.GetUserId()
	connID := uuid.NewString()
	startedAt := time.Now()

	ch := s.hub.Subscribe(notebookID, connID, 64)
	defer s.hub.Unsubscribe(notebookID, connID)

	ctx := stream.Context()
	logger.Info(ctx, "grpc.SubscribeNotebook", "stage", "open", "user_id", userID, "notebook_id", notebookID, "conn_id", connID)

	var sent int
	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "grpc.SubscribeNotebook",
				"stage", "close",
				"reason", "context_done",
				"user_id", userID,
				"notebook_id", notebookID,
				"conn_id", connID,
				"events_sent", sent,
				"duration", time.Since(startedAt).String(),
			)
			return nil
		case event, ok := <-ch:
			if !ok {
				logger.Info(ctx, "grpc.SubscribeNotebook",
					"stage", "close",
					"reason", "channel_closed",
					"user_id", userID,
					"notebook_id", notebookID,
					"conn_id", connID,
					"events_sent", sent,
					"duration", time.Since(startedAt).String(),
				)
				return nil
			}
			if err := stream.Send(event); err != nil {
				logger.Error(ctx, "grpc.SubscribeNotebook",
					"stage", "send_failed",
					"error", err,
					"user_id", userID,
					"notebook_id", notebookID,
					"conn_id", connID,
					"events_sent", sent,
					"duration", time.Since(startedAt).String(),
				)
				return err
			}
			sent++
		}
	}
}

func notebookToProto(nb *domain.Notebook) *pb.NotebookInfo {
	if nb == nil {
		return nil
	}
	info := &pb.NotebookInfo{
		Id:        nb.ID,
		OwnerId:   nb.OwnerID,
		OwnerName: nb.OwnerUsername,
		Title:     nb.Title,
		IsPublic:  nb.IsPublic,
		CreatedAt: nb.CreatedAt.Unix(),
		UpdatedAt: nb.UpdatedAt.Unix(),
	}
	if len(nb.Blocks) > 0 {
		info.Blocks = make([]*pb.BlockInfo, len(nb.Blocks))
		for i := range nb.Blocks {
			info.Blocks[i] = blockToProto(&nb.Blocks[i])
		}
	}
	return info
}

func (s *Server) SaveBlockOutputs(ctx context.Context, req *pb.SaveBlockOutputsRequest) (*pb.SaveBlockOutputsResponse, error) {
	outputs := make([]domain.BlockOutput, len(req.GetOutputs()))
	for i, o := range req.GetOutputs() {
		outputs[i] = domain.BlockOutput{
			BlockID:    req.GetBlockId(),
			Position:   int(o.GetPosition()),
			OutputType: o.GetOutputType(),
			Content:    o.GetContent(),
		}
	}
	if err := s.notebookUC.SaveBlockOutputs(ctx, req.GetBlockId(), outputs); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.SaveBlockOutputsResponse{}, nil
}

func (s *Server) ImportNotebook(ctx context.Context, req *pb.ImportNotebookRequest) (*pb.NotebookResponse, error) {
	blocks := make([]domain.Block, len(req.GetBlocks()))
	for i, b := range req.GetBlocks() {
		blocks[i] = domain.Block{
			Type:     b.GetType(),
			Language: b.GetLanguage(),
			Content:  b.GetContent(),
			Position: int(b.GetPosition()),
		}
		for _, o := range b.GetOutputs() {
			blocks[i].Outputs = append(blocks[i].Outputs, domain.BlockOutput{
				Position:   int(o.GetPosition()),
				OutputType: o.GetOutputType(),
				Content:    o.GetContent(),
			})
		}
	}
	nb, err := s.notebookUC.ImportNotebook(ctx, req.GetUserId(), req.GetTitle(), blocks)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.NotebookResponse{Notebook: notebookToProto(nb)}, nil
}

func (s *Server) ReorderBlocks(ctx context.Context, req *pb.ReorderBlocksRequest) (*pb.ReorderBlocksResponse, error) {
	if err := s.notebookUC.ReorderBlocks(ctx, req.GetUserId(), req.GetNotebookId(), req.GetBlockIds()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.ReorderBlocksResponse{}, nil
}

func blockToProto(b *domain.Block) *pb.BlockInfo {
	if b == nil {
		return nil
	}
	info := &pb.BlockInfo{
		Id:         b.ID,
		NotebookId: b.NotebookID,
		Type:       b.Type,
		Language:   b.Language,
		Content:    b.Content,
		Position:   int32(b.Position), //nolint:gosec // block position fits int32
		CreatedAt:  b.CreatedAt.Unix(),
		UpdatedAt:  b.UpdatedAt.Unix(),
	}
	if b.ExecutionCount != nil {
		v := int32(*b.ExecutionCount) //nolint:gosec // execution count fits int32
		info.ExecutionCount = &v
	}
	if len(b.Outputs) > 0 {
		info.Outputs = make([]*pb.BlockOutputInfo, len(b.Outputs))
		for i, o := range b.Outputs {
			info.Outputs[i] = &pb.BlockOutputInfo{
				Id:         o.ID,
				BlockId:    o.BlockID,
				Position:   int32(o.Position), //nolint:gosec // output position fits int32
				OutputType: o.OutputType,
				Content:    o.Content,
				CreatedAt:  o.CreatedAt.Unix(),
			}
		}
	}
	return info
}

func (s *Server) GetUserStats(ctx context.Context, req *pb.GetUserNotebookStatsRequest) (*pb.GetUserNotebookStatsResponse, error) {
	nbCount, blockCount, totalExec, err := s.notebookUC.GetUserResourceStats(ctx, req.GetUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.GetUserNotebookStatsResponse{
		NotebookCount:   nbCount,
		BlockCount:      blockCount,
		TotalExecutions: totalExec,
	}, nil
}
