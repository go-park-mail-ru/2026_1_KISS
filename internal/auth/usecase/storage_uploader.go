package usecase

import (
	"context"
	"fmt"
	"io"

	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type StorageUploader struct {
	client pb.StorageServiceClient
}

func NewStorageUploader(client pb.StorageServiceClient) *StorageUploader {
	return &StorageUploader{client: client}
}

func (u *StorageUploader) Upload(ctx context.Context, ownerID int64, category, filename string, data io.Reader, size int64) (string, error) {
	stream, err := u.client.UploadFile(ctx)
	if err != nil {
		return "", fmt.Errorf("open upload stream: %w", err)
	}

	buf := make([]byte, 64*1024)
	first := true
	for {
		n, readErr := data.Read(buf)
		if n > 0 {
			chunk := &pb.UploadFileChunk{Data: buf[:n]}
			if first {
				chunk.OwnerId = ownerID
				chunk.Filename = filename
				chunk.FileSize = size
				chunk.Category = category
				first = false
			}
			if err := stream.Send(chunk); err != nil {
				return "", fmt.Errorf("send chunk: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("read data: %w", readErr)
		}
	}

	if first {
		if err := stream.Send(&pb.UploadFileChunk{
			OwnerId:  ownerID,
			Filename: filename,
			FileSize: size,
			Category: category,
			Data:     []byte{},
		}); err != nil {
			return "", fmt.Errorf("send empty chunk: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", fmt.Errorf("close upload: %w", err)
	}

	return resp.GetFile().GetUrl(), nil
}

func (u *StorageUploader) Delete(ctx context.Context, url string) error {
	if url == "" {
		return nil
	}
	_, err := u.client.DeleteFileByURL(ctx, &pb.DeleteFileByURLRequest{Url: url})
	return err
}
