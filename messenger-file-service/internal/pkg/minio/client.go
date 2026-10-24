package minio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/MrEsbens/messenger-file-service/internal/pkg/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	*minio.Client
	endpoint string
	useSSL   bool
}

type MinIOClient interface {
	GetPresignedPutURL(ctx context.Context, bucketType storage.BucketType, objectKey string, expiry time.Duration) (string, error)
	GetPresignedGetURL(ctx context.Context, bucketType storage.BucketType, objectKey string, expiry time.Duration) (string, error)
	StatObject(ctx context.Context, bucket, key string) error
	RemoveObject(ctx context.Context, bucket, key string, opts minio.RemoveObjectOptions) error
	Endpoint() string
}

var _ MinIOClient = (*Client)(nil)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string
}

func NewClient(cfg Config) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &Client{
		Client:   client,
		endpoint: cfg.Endpoint,
		useSSL:   cfg.UseSSL,
	}, nil
}

func (c *Client) EnsureBucket(ctx context.Context, bucketType storage.BucketType) error {
	bucketName := bucketType.Name()
	exists, err := c.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !exists {
		return c.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}
	return nil
}

func (c *Client) Upload(ctx context.Context, bucketType storage.BucketType, objectKey string, reader io.Reader, objectSize int64, contentType string) error {
	bucketName := bucketType.Name()

	_, err := c.PutObject(ctx, bucketName, objectKey, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s: %w", objectKey, bucketName, err)
	}
	return nil
}

func (c *Client) GetPublicURL(bucketType storage.BucketType, objectKey string) string {
	bucketName := bucketType.Name()
	protocol := "http"
	if c.useSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", protocol, c.endpoint, bucketName, objectKey)
}

func (c *Client) Delete(ctx context.Context, bucketType storage.BucketType, objectKey string) error {
	return c.RemoveObject(ctx, bucketType.Name(), objectKey, minio.RemoveObjectOptions{})
}

func (c *Client) GetPresignedPutURL(ctx context.Context, bucketType storage.BucketType, objectKey string, expiry time.Duration) (string, error) {
	bucketName := bucketType.Name()

	url, err := c.PresignedPutObject(ctx, bucketName, objectKey, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}

	return url.String(), nil
}

func (c *Client) GetPresignedGetURL(ctx context.Context, bucketType storage.BucketType, objectKey string, expiry time.Duration) (string, error) {
	bucketName := bucketType.Name()

	url, err := c.PresignedGetObject(ctx, bucketName, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned GET URL: %w", err)
	}

	return url.String(), nil
}

func (c *Client) StatObject(ctx context.Context, bucketName, objectKey string) error {
	_, err := c.Client.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{})
	return err
}

func (c *Client) RemoveObject(ctx context.Context, bucket, objectKey string, opts minio.RemoveObjectOptions) error {
	return c.Client.RemoveObject(ctx, bucket, objectKey, opts)
}

func (c *Client) Endpoint() string {
	return c.endpoint
}
