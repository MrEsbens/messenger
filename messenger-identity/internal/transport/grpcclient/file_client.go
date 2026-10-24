package grpcclient

import (
	"context"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type FileClientInterface interface {
	GetUploadURL(ctx context.Context, req *filev1.GetUploadURLRequest) (*filev1.GetUploadURLResponse, error)
	ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*emptypb.Empty, error)
	GetDownloadURL(ctx context.Context, req *filev1.GetDownloadURLRequest) (*filev1.GetDownloadURLResponse, error)
	GetFileByOwner(ctx context.Context, req *filev1.GetFileByOwnerRequest) (*filev1.GetFileByOwnerResponse, error)
	Close() error
}

var _ FileClientInterface = (*FileClient)(nil)

type FileClient struct {
	conn   *grpc.ClientConn
	client filev1.FileServiceClient
}

func NewFileClient(endpoint string) (*FileClient, error) {
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return nil, err
	}

	return &FileClient{
		conn:   conn,
		client: filev1.NewFileServiceClient(conn),
	}, nil
}

func (c *FileClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *FileClient) GetUploadURL(ctx context.Context, req *filev1.GetUploadURLRequest) (*filev1.GetUploadURLResponse, error) {
	return c.client.GetUploadURL(ctx, req)
}

func (c *FileClient) GetDownloadURL(ctx context.Context, req *filev1.GetDownloadURLRequest) (*filev1.GetDownloadURLResponse, error) {
	return c.client.GetDownloadURL(ctx, req)
}

func (c *FileClient) ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*emptypb.Empty, error) {
	return c.client.ConfirmUpload(ctx, req)
}

func (c *FileClient) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*emptypb.Empty, error) {
	return c.client.DeleteFile(ctx, req)
}

func (c *FileClient) GetFileByOwner(ctx context.Context, req *filev1.GetFileByOwnerRequest) (*filev1.GetFileByOwnerResponse, error) {
	return c.client.GetFileByOwner(ctx, req)
}
