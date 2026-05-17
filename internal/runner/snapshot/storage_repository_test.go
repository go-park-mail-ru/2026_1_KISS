package snapshot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/snapshot"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
	"go.uber.org/mock/gomock"
)

func newRepo(ctrl *gomock.Controller) (snapshot.Repository, *mocks.MockStorageServiceClient) {
	client := mocks.NewMockStorageServiceClient(ctrl)
	return snapshot.NewStorageRepository(client, 10*1024*1024), client
}

func TestExists_NoFiles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, client := newRepo(ctrl)

	nb := int64(1)
	client.EXPECT().ListFiles(gomock.Any(), &pb.ListFilesRequest{
		UserId:     10,
		Category:   "sessions",
		NotebookId: &nb,
		Limit:      1,
	}).Return(&pb.ListFilesResponse{Files: nil}, nil)

	ok, err := repo.Exists(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false, got true")
	}
}

func TestExists_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, client := newRepo(ctrl)

	nb := int64(1)
	client.EXPECT().ListFiles(gomock.Any(), &pb.ListFilesRequest{
		UserId:     10,
		Category:   "sessions",
		NotebookId: &nb,
		Limit:      1,
	}).Return(&pb.ListFilesResponse{Files: []*pb.FileInfo{{Id: "f-1"}}}, nil)

	ok, err := repo.Exists(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
}

func TestExists_ListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, client := newRepo(ctrl)

	nb := int64(1)
	client.EXPECT().ListFiles(gomock.Any(), &pb.ListFilesRequest{
		UserId:     10,
		Category:   "sessions",
		NotebookId: &nb,
		Limit:      1,
	}).Return(nil, errors.New("grpc error"))

	_, err := repo.Exists(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_NoFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, client := newRepo(ctrl)

	nb := int64(1)
	client.EXPECT().ListFiles(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *pb.ListFilesRequest, _ ...interface{}) (*pb.ListFilesResponse, error) {
			if req.NotebookId == nil || *req.NotebookId != nb {
				t.Errorf("unexpected notebook_id in request")
			}
			return &pb.ListFilesResponse{Files: nil}, nil
		})

	err := repo.Delete(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, client := newRepo(ctrl)

	nb := int64(1)
	client.EXPECT().ListFiles(gomock.Any(), &pb.ListFilesRequest{
		UserId:     10,
		Category:   "sessions",
		NotebookId: &nb,
		Limit:      1,
	}).Return(&pb.ListFilesResponse{Files: nil}, nil)

	_, _, err := repo.Load(context.Background(), 1, 10)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_WithFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo, client := newRepo(ctrl)

	nb := int64(1)
	client.EXPECT().ListFiles(gomock.Any(), &pb.ListFilesRequest{
		UserId:     10,
		Category:   "sessions",
		NotebookId: &nb,
		Limit:      1,
	}).Return(&pb.ListFilesResponse{Files: []*pb.FileInfo{{Id: "f-123"}}}, nil)

	client.EXPECT().DeleteFile(gomock.Any(), &pb.DeleteFileRequest{
		FileId: "f-123",
		UserId: 10,
	}).Return(&pb.DeleteFileResponse{}, nil)

	err := repo.Delete(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSave_ExceedsMaxSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mocks.NewMockStorageServiceClient(ctrl)
	repo := snapshot.NewStorageRepository(client, 5) // 5 byte limit

	err := repo.Save(context.Background(), 1, 10, []byte("hello world")) // 11 bytes
	if err == nil {
		t.Fatal("expected error for oversized snapshot")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}
