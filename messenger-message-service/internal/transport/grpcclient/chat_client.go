package grpcclient

import (
	"context"
	"fmt"
	"time"

	chatsv1 "github.com/MrEsbens/messenger-chats-service/api/chats/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type ChatClientInterface interface {
	IsMember(ctx context.Context, chatID, userID string) (*IsMemberResult, error)
	Close() error
}

type IsMemberResult struct {
	IsMember bool
	Role     chatsv1.MembershipRole
}

type ChatClient struct {
	conn   *grpc.ClientConn
	client chatsv1.ChatServiceClient
}

func NewChatClient(endpoint string) (*ChatClient, error) {
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
		return nil, fmt.Errorf("failed to connect to chats service: %w", err)
	}

	return &ChatClient{
		conn:   conn,
		client: chatsv1.NewChatServiceClient(conn),
	}, nil
}

func (c *ChatClient) IsMember(ctx context.Context, chatID, userID string) (*IsMemberResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.client.IsMember(ctx, &chatsv1.IsMemberRequest{
		ChatId: chatID,
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}

	// 🔧 Возвращаем proto-тип, не domain!
	return &IsMemberResult{
		IsMember: resp.IsMember,
		Role:     resp.Role,
	}, nil
}

func (c *ChatClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
