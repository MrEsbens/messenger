package grpcclient

import (
	"context"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	moderationpb "github.com/MrEsbens/messenger-moderation-service/api/go/moderation/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ModerationClientInterface interface {
	ModerateText(ctx context.Context, text, userID, serverID string, filters *domain.ModerationFilters) (*domain.ModerationResult, error)
	Close() error
}

type moderationClient struct {
	conn    *grpc.ClientConn
	client  moderationpb.ModerationServiceClient
	timeout time.Duration
}

func NewModerationClient(endpoint string, timeout time.Duration) (ModerationClientInterface, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("moderation service endpoint is required")
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: заменить на mTLS в проде
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true), // Ждать готовности сервиса перед вызовом
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create moderation client at %s: %w", endpoint, err)
	}

	return &moderationClient{
		conn:    conn,
		client:  moderationpb.NewModerationServiceClient(conn),
		timeout: timeout,
	}, nil
}

func (c *moderationClient) ModerateText(
	ctx context.Context,
	text, userID, serverID string,
	filters *domain.ModerationFilters,
) (*domain.ModerationResult, error) {
	if text == "" {
		return &domain.ModerationResult{Allowed: true}, nil
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	protoFilters := domainFiltersToProto(filters)

	resp, err := c.client.ModerateText(callCtx, &moderationpb.ModerateTextRequest{
		Text:     text,
		UserId:   userID,
		ServerId: serverID,
		Filters:  protoFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("moderation check failed: %w", err)
	}

	return protoToDomainResult(resp), nil
}

func (c *moderationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func domainFiltersToProto(filters *domain.ModerationFilters) *moderationpb.ModerateFilters {
	if filters == nil {
		return &moderationpb.ModerateFilters{
			ProfanityAction:  "delete",
			ToxicityAction:   "delete",
			NsfwTextAction:   "delete",
			HateSpeechAction: "delete",
			MlScoreThreshold: 0.7,
		}
	}

	return &moderationpb.ModerateFilters{
		ProfanityAction:  filters.ProfanityAction,
		ToxicityAction:   filters.ToxicityAction,
		NsfwTextAction:   filters.NsfwTextAction,
		HateSpeechAction: filters.HateSpeechAction,
		MlScoreThreshold: float32(filters.MLScoreThreshold),
	}
}

func protoToDomainResult(proto *moderationpb.ModerateTextResponse) *domain.ModerationResult {
	if proto == nil {
		return &domain.ModerationResult{Allowed: true}
	}

	violations := make([]domain.Violation, 0, len(proto.Violations))
	for _, v := range proto.Violations {
		violations = append(violations, domain.Violation{
			Type:    v.Type,
			Message: v.Message,
			Score:   float32(v.Score),
		})
	}

	return &domain.ModerationResult{
		Allowed:    proto.Allowed,
		Violations: violations,
	}
}
