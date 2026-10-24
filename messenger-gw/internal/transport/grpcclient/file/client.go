package file

import (
	"context"
	"fmt"
	"strings"
	"time"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type FileClientInterface interface {
	GetUploadURL(ctx context.Context, req *GetUploadURLRequest) (*GetUploadURLResponse, error)
	ConfirmUpload(ctx context.Context, req *ConfirmUploadRequest) error
	GetDownloadURL(ctx context.Context, req *GetDownloadURLRequest) (*GetDownloadURLResponse, error)
	DeleteFile(ctx context.Context, req *DeleteFileRequest) error
	GetFileByOwner(ctx context.Context, req *GetFileByOwnerRequest) (*GetFileByOwnerResponse, error)
	GetFilesByOwner(ctx context.Context, req *GetFilesByOwnerRequest) (*GetFilesByOwnerResponse, error)
	Close() error
}

type fileClient struct {
	client  filev1.FileServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewFileClient(endpoint string, timeout time.Duration) (FileClientInterface, error) {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "grpc://")

	if endpoint == "" {
		return nil, fmt.Errorf("empty file service endpoint")
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithReturnConnectionError(),
	}

	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to file service: %w", err)
	}

	return &fileClient{
		client:  filev1.NewFileServiceClient(conn),
		conn:    conn,
		timeout: timeout,
	}, nil
}

func (c *fileClient) GetUploadURL(ctx context.Context, req *GetUploadURLRequest) (*GetUploadURLResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetUploadURL(callCtx, &filev1.GetUploadURLRequest{
		OwnerId:    req.OwnerID.String(),
		OwnerType:  req.OwnerType,
		BucketType: req.BucketType,
		FileName:   req.FileName,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetUploadURL failed: %w", err)
	}

	return &GetUploadURLResponse{
		FileID:    uuid.MustParse(resp.FileId),
		UploadURL: resp.UploadUrl,
		ObjectKey: resp.ObjectKey,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	}, nil
}

func (c *fileClient) ConfirmUpload(ctx context.Context, req *ConfirmUploadRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.ConfirmUpload(callCtx, &filev1.ConfirmUploadRequest{
		FileId: req.FileID.String(),
	})
	return err
}

func (c *fileClient) GetDownloadURL(ctx context.Context, req *GetDownloadURLRequest) (*GetDownloadURLResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetDownloadURL(callCtx, &filev1.GetDownloadURLRequest{
		FileId:        req.FileID.String(),
		ExpirySeconds: req.ExpirySeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetDownloadURL failed: %w", err)
	}

	return &GetDownloadURLResponse{
		DownloadURL: resp.DownloadUrl,
		ExpiresAt:   resp.ExpiresAt.AsTime(),
	}, nil
}

func (c *fileClient) DeleteFile(ctx context.Context, req *DeleteFileRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.DeleteFile(callCtx, &filev1.DeleteFileRequest{
		FileId:  req.FileID.String(),
		OwnerId: req.OwnerID.String(),
	})
	return err
}

func (c *fileClient) GetFileByOwner(ctx context.Context, req *GetFileByOwnerRequest) (*GetFileByOwnerResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetFileByOwner(callCtx, &filev1.GetFileByOwnerRequest{
		OwnerId:    req.OwnerID.String(),
		OwnerType:  req.OwnerType,
		BucketType: req.BucketType,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetFileByOwner failed: %w", err)
	}

	return &GetFileByOwnerResponse{
		FileID:    uuid.MustParse(resp.FileId),
		ObjectKey: resp.ObjectKey,
		PublicURL: resp.PublicUrl,
		MimeType:  resp.MimeType,
		FileSize:  resp.FileSize,
		FileName:  resp.FileName,
	}, nil
}

func (c *fileClient) GetFilesByOwner(ctx context.Context, req *GetFilesByOwnerRequest) (*GetFilesByOwnerResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetFilesByOwner(callCtx, &filev1.GetFilesByOwnerRequest{
		OwnerId:    req.OwnerID.String(),
		OwnerType:  req.OwnerType,
		BucketType: req.BucketType,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetFilesByOwner failed: %w", err)
	}

	files := make([]FileInfo, 0, len(resp.Files))
	for _, f := range resp.Files {
		files = append(files, FileInfo{
			FileID:    uuid.MustParse(f.FileId),
			ObjectKey: f.ObjectKey,
			PublicURL: f.PublicUrl,
			MimeType:  f.MimeType,
			FileSize:  f.FileSize,
			FileName:  f.FileName,
		})
	}

	return &GetFilesByOwnerResponse{Files: files}, nil
}

func (c *fileClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
