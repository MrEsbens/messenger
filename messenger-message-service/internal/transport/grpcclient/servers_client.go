package grpcclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	serverspb "github.com/MrEsbens/messenger-servers-service/api/servers/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServersClientInterface interface {
	GetModerationConfig(ctx context.Context, serverID string) (*domain.ModerationConfig, error)
	IsMember(ctx context.Context, serverID, userID string) (bool, error)
	BanMember(ctx context.Context, serverID, userID, executorID string) error
	MuteMember(ctx context.Context, serverID, userID, executorID string, duration time.Duration) error
	Close() error
}

type serversClient struct {
	client  serverspb.ServersServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewServersClient(endpoint string, timeout time.Duration) (ServersClientInterface, error) {
	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to servers service: %w", err)
	}

	return &serversClient{
		client:  serverspb.NewServersServiceClient(conn),
		conn:    conn,
		timeout: timeout,
	}, nil
}

func (c *serversClient) GetModerationConfig(ctx context.Context, serverID string) (*domain.ModerationConfig, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetModerationConfig(callCtx, &serverspb.GetModerationConfigRequest{
		ServerId: serverID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get moderation config: %w", err)
	}

	// 🔧 Используем string, как в твоем domain.ModerationConfig
	return &domain.ModerationConfig{
		ServerID:               resp.ServerId, // ← просто строка, не uuid.UUID
		ProfanityFilterAction:  resp.ProfanityFilterAction,
		ToxicityFilterAction:   resp.ToxicityFilterAction,
		NsfwTextFilterAction:   resp.NsfwTextFilterAction,
		HateSpeechFilterAction: resp.HateSpeechFilterAction,
	}, nil
}

func (c *serversClient) IsMember(ctx context.Context, serverID, userID string) (bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.GetMember(callCtx, &serverspb.GetMemberRequest{
		ServerId: serverID,
		UserId:   userID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not a member") || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return true, nil
}

func (c *serversClient) BanMember(ctx context.Context, serverID, userID, executorID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.BanMember(callCtx, &serverspb.BanMemberRequest{
		ServerId: serverID,
		UserId:   userID,
		BannedBy: executorID,
	})
	if err != nil {
		return fmt.Errorf("failed to ban member: %w", err)
	}
	return nil
}

func (c *serversClient) MuteMember(ctx context.Context, serverID, userID, executorID string, duration time.Duration) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	durationHours := int64(duration.Hours())
	if durationHours < 1 {
		durationHours = 24
	}

	_, err := c.client.MuteMember(callCtx, &serverspb.MuteMemberRequest{
		ServerId:      serverID,
		UserId:        userID,
		MutedBy:       executorID,
		DurationHours: durationHours,
	})
	if err != nil {
		return fmt.Errorf("failed to mute member: %w", err)
	}
	return nil
}

func (c *serversClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
