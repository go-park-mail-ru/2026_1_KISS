package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

const (
	snapshotCategory = "sessions"
	snapshotFilename = "snapshot.dill"
	uploadChunkSize  = 32 * 1024
)

type storageRepository struct {
	client  pb.StorageServiceClient
	maxSize int64
}

func NewStorageRepository(client pb.StorageServiceClient, maxSize int64) Repository {
	return &storageRepository{client: client, maxSize: maxSize}
}

func (r *storageRepository) Save(ctx context.Context, notebookID, userID int64, data []byte) error {
	logger.Info(ctx, "snapshot.Save", "notebook_id", notebookID, "user_id", userID, "size", len(data))

	if r.maxSize > 0 && int64(len(data)) > r.maxSize {
		return fmt.Errorf("%w: snapshot size %d exceeds limit %d", domain.ErrInvalidInput, len(data), r.maxSize)
	}

	// Удаляем старый снапшот
	if fileID, err := r.findFileID(ctx, notebookID, userID); err == nil && fileID != "" {
		if _, delErr := r.client.DeleteFile(ctx, &pb.DeleteFileRequest{FileId: fileID, UserId: userID}); delErr != nil {
			logger.Error(ctx, "snapshot.Save: delete old", "error", delErr)
		}
	}

	stream, err := r.client.UploadFile(ctx)
	if err != nil {
		return fmt.Errorf("open upload stream: %w", err)
	}

	reader := bytes.NewReader(data)
	buf := make([]byte, uploadChunkSize)
	first := true
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			chunk := &pb.UploadFileChunk{Data: buf[:n]}
			if first {
				chunk.OwnerId = userID
				chunk.Filename = snapshotFilename
				chunk.FileSize = int64(len(data))
				chunk.Category = snapshotCategory
				nbID := notebookID
				chunk.NotebookId = &nbID
				first = false
			}
			if sendErr := stream.Send(chunk); sendErr != nil {
				return fmt.Errorf("send chunk: %w", sendErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read data: %w", readErr)
		}
	}

	if _, err := stream.CloseAndRecv(); err != nil {
		return fmt.Errorf("close upload stream: %w", err)
	}
	return nil
}

func (r *storageRepository) Load(ctx context.Context, notebookID, userID int64) ([]byte, Metadata, error) {
	logger.Info(ctx, "snapshot.Load", "notebook_id", notebookID, "user_id", userID)

	fileID, err := r.findFileID(ctx, notebookID, userID)
	if err != nil {
		return nil, Metadata{}, err
	}
	if fileID == "" {
		return nil, Metadata{}, domain.ErrNotFound
	}

	stream, err := r.client.DownloadFile(ctx, &pb.DownloadFileRequest{FileId: fileID, UserId: userID})
	if err != nil {
		return nil, Metadata{}, fmt.Errorf("open download stream: %w", err)
	}

	var buf []byte
	var totalSize int64
	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return nil, Metadata{}, fmt.Errorf("recv chunk: %w", recvErr)
		}
		if chunk.TotalSize > 0 {
			totalSize = chunk.TotalSize
		}
		buf = append(buf, chunk.Data...)
	}

	meta := Metadata{
		FileID:    fileID,
		SavedAt:   time.Now(),
		SizeBytes: totalSize,
	}
	return buf, meta, nil
}

func (r *storageRepository) Delete(ctx context.Context, notebookID, userID int64) error {
	fileID, err := r.findFileID(ctx, notebookID, userID)
	if err != nil || fileID == "" {
		return nil
	}
	_, err = r.client.DeleteFile(ctx, &pb.DeleteFileRequest{FileId: fileID, UserId: userID})
	return err
}

func (r *storageRepository) Exists(ctx context.Context, notebookID, userID int64) (bool, error) {
	fileID, err := r.findFileID(ctx, notebookID, userID)
	if err != nil {
		return false, err
	}
	return fileID != "", nil
}

func (r *storageRepository) findFileID(ctx context.Context, notebookID, userID int64) (string, error) {
	nb := notebookID
	resp, err := r.client.ListFiles(ctx, &pb.ListFilesRequest{
		UserId:     userID,
		Category:   snapshotCategory,
		NotebookId: &nb,
		Limit:      1,
	})
	if err != nil {
		return "", fmt.Errorf("list snapshots: %w", err)
	}
	if len(resp.Files) == 0 {
		return "", nil
	}
	return resp.Files[0].Id, nil
}
