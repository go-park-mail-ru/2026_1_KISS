package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

type Server struct {
	pb.UnimplementedNotebookServiceServer
	notebookUC usecase.NotebookService
	blockRepo  repository.BlockRepository
}

func NewServer(notebookUC usecase.NotebookService, blockRepo repository.BlockRepository) *Server {
	return &Server{notebookUC: notebookUC, blockRepo: blockRepo}
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

func notebookToProto(nb *domain.Notebook) *pb.NotebookInfo {
	if nb == nil {
		return nil
	}
	info := &pb.NotebookInfo{
		Id:        nb.ID,
		OwnerId:   nb.OwnerID,
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
	return info
}
