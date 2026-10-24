package grpcclient

import (
	"context"
	"fmt"
	"time"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/backoff"
)

type FileClientInterface interface {
	GetFileByOwner(ctx context.Context, ownerID, ownerType, bucketType string) (*FileMetadata, error)
	GetFilesByOwner(ctx context.Context, ownerID, ownerType, bucketType string) ([]*FileMetadata, error)
	Close() error
}

// FileMetadata — наша доменная обёртка над proto-ответом
// Чтобы не зависеть от конкретных полей proto в бизнес-логике
type FileMetadata struct {
	FileID    string
	ObjectKey string
	PublicURL string
	MimeType  string
	FileSize  int64
	FileName  string
}

type FileClient struct {
	conn   *grpc.ClientConn
	client filev1.FileServiceClient
}

func NewFileClient(endpoint string) (*FileClient, error) {
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				Multiplier: 1.6,
				MaxDelay:   5 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
			grpc.MaxCallRecvMsgSize(1024*1024),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to file service: %w", err)
	}

	return &FileClient{
		conn:   conn,
		client: filev1.NewFileServiceClient(conn),
	}, nil
}

// GetFileByOwner получает метаданные файла по владельцу
// Возвращает nil, nil если файл не найден (это ок — чат может быть без аватара)
func (c *FileClient) GetFileByOwner(ctx context.Context, ownerID, ownerType, bucketType string) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.client.GetFileByOwner(ctx, &filev1.GetFileByOwnerRequest{
		OwnerId:    ownerID,
		OwnerType:  ownerType,
		BucketType: bucketType,
	})
	if err != nil {
		return nil, nil
	}

	return &FileMetadata{
		FileID:    resp.FileId,
		ObjectKey: resp.ObjectKey,
		PublicURL: resp.PublicUrl,
		MimeType:  resp.MimeType,
		FileSize:  resp.FileSize,
		FileName:  resp.FileName,
	}, nil
}

// GetFilesByOwner получает список файлов владельца
func (c *FileClient) GetFilesByOwner(ctx context.Context, ownerID, ownerType, bucketType string) ([]*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.client.GetFilesByOwner(ctx, &filev1.GetFilesByOwnerRequest{
		OwnerId:    ownerID,
		OwnerType:  ownerType,
		BucketType: bucketType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get files by owner: %w", err)
	}

	// 🔧 Конвертируем список FileInfo в нашу доменную структуру
	files := make([]*FileMetadata, 0, len(resp.Files))
	for _, f := range resp.Files {
		files = append(files, &FileMetadata{
			FileID:    f.FileId,
			ObjectKey: f.ObjectKey,
			PublicURL: f.PublicUrl,
			MimeType:  f.MimeType,
			FileSize:  f.FileSize,
			FileName:  f.FileName,
		})
	}

	return files, nil
}

func (c *FileClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}