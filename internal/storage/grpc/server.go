package grpc

import (
	"bytes"
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/storage/usecase"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

const defaultPageSize = 20

type Server struct {
	pb.UnimplementedStorageServiceServer
	uc *usecase.StorageUsecase
}

func NewServer(uc *usecase.StorageUsecase) *Server {
	return &Server{uc: uc}
}

func (s *Server) UploadFile(stream pb.StorageService_UploadFileServer) error {
	var buf bytes.Buffer
	var ownerID int64
	var filename string
	var fileSize int64
	var category string
	var notebookID *int64

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive chunk: %v", err)
		}
		if ownerID == 0 {
			ownerID = chunk.GetOwnerId()
			filename = chunk.GetFilename()
			fileSize = chunk.GetFileSize()
			category = chunk.GetCategory()
			if chunk.NotebookId != nil {
				nbID := chunk.GetNotebookId()
				notebookID = &nbID
			}
		}
		buf.Write(chunk.GetData())
	}

	if ownerID == 0 {
		return status.Error(codes.InvalidArgument, "owner_id is required")
	}

	file, err := s.uc.UploadFile(
		stream.Context(), ownerID, domain.FileCategory(category),
		filename, bytes.NewReader(buf.Bytes()), fileSize, notebookID,
	)
	if err != nil {
		return grpcutil.DomainToGRPCError(err)
	}

	return stream.SendAndClose(&pb.FileResponse{File: fileToProto(file)})
}

func (s *Server) GetFile(ctx context.Context, req *pb.GetFileRequest) (*pb.FileResponse, error) {
	file, err := s.uc.GetFile(ctx, req.GetFileId(), req.GetUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.FileResponse{File: fileToProto(file)}, nil
}

func (s *Server) ListFiles(ctx context.Context, req *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultPageSize
	}
	files, total, err := s.uc.ListFiles(ctx, req.GetUserId(), req.GetCategory(), limit, int(req.GetOffset()))
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	pbFiles := make([]*pb.FileInfo, len(files))
	for i := range files {
		pbFiles[i] = fileToProto(&files[i])
	}
	return &pb.ListFilesResponse{Files: pbFiles, Total: int32(total)}, nil //nolint:gosec
}

func (s *Server) DeleteFile(ctx context.Context, req *pb.DeleteFileRequest) (*pb.DeleteFileResponse, error) {
	if err := s.uc.DeleteFile(ctx, req.GetFileId(), req.GetUserId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.DeleteFileResponse{}, nil
}

func (s *Server) GetStorageStats(ctx context.Context, _ *pb.GetStorageStatsRequest) (*pb.GetStorageStatsResponse, error) {
	stats, err := s.uc.GetStats(ctx)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	byCategory := make(map[string]int64, len(stats.FilesByCategory))
	sizeByCategory := make(map[string]int64, len(stats.SizeByCategory))
	for k, v := range stats.FilesByCategory {
		byCategory[string(k)] = v
	}
	for k, v := range stats.SizeByCategory {
		sizeByCategory[string(k)] = v
	}
	return &pb.GetStorageStatsResponse{
		TotalFiles:      stats.TotalFiles,
		TotalSizeBytes:  stats.TotalSizeBytes,
		FilesByCategory: byCategory,
		SizeByCategory:  sizeByCategory,
	}, nil
}

func (s *Server) AdminListFiles(ctx context.Context, req *pb.AdminListFilesRequest) (*pb.ListFilesResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultPageSize
	}
	files, total, err := s.uc.AdminListFiles(ctx, req.GetCategory(), req.GetOwnerId(), limit, int(req.GetOffset()))
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	pbFiles := make([]*pb.FileInfo, len(files))
	for i := range files {
		pbFiles[i] = fileToProto(&files[i])
	}
	return &pb.ListFilesResponse{Files: pbFiles, Total: int32(total)}, nil //nolint:gosec
}

func (s *Server) DeleteFileByURL(ctx context.Context, req *pb.DeleteFileByURLRequest) (*pb.DeleteFileResponse, error) {
	if err := s.uc.DeleteFileByURL(ctx, req.GetUrl()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.DeleteFileResponse{}, nil
}

func (s *Server) GetUserStorageStats(ctx context.Context, req *pb.GetUserStorageStatsRequest) (*pb.GetStorageStatsResponse, error) {
	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	stats, err := s.uc.GetUserStorageStats(ctx, req.GetUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}

	filesByCat := make(map[string]int64, len(stats.FilesByCategory))
	sizeByCat := make(map[string]int64, len(stats.SizeByCategory))
	for k, v := range stats.FilesByCategory {
		filesByCat[string(k)] = v
	}
	for k, v := range stats.SizeByCategory {
		sizeByCat[string(k)] = v
	}

	return &pb.GetStorageStatsResponse{
		TotalFiles:      stats.TotalFiles,
		TotalSizeBytes:  stats.TotalSizeBytes,
		FilesByCategory: filesByCat,
		SizeByCategory:  sizeByCat,
	}, nil
}

func fileToProto(f *domain.File) *pb.FileInfo {
	if f == nil {
		return nil
	}
	info := &pb.FileInfo{
		Id:        f.ID,
		OwnerId:   f.OwnerID,
		Category:  string(f.Category),
		Filename:  f.Filename,
		Url:       f.URL,
		MimeType:  f.MIMEType,
		Size:      f.Size,
		CreatedAt: f.CreatedAt.Unix(),
	}
	if f.NotebookID != nil {
		info.NotebookId = f.NotebookID
	}
	return info
}
