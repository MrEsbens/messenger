package messages

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/data"
	messagesv1 "github.com/MrEsbens/messenger-message-service/api/messages/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── Интерфейс ─────────────────────────────────────────────────────────────

type MessagesClientInterface interface {
	SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)
	UpdateMessageStatus(ctx context.Context, req *UpdateStatusRequest) error
	EditMessage(ctx context.Context, req *EditMessageRequest) error
	DeleteMessage(ctx context.Context, req *DeleteMessageRequest) error
	GetMessages(ctx context.Context, req *GetMessagesRequest) (*GetMessagesResponse, error)
	GetMessageStatuses(ctx context.Context, req *GetMessageStatusesRequest) (*GetMessageStatusesResponse, error)
	GetMessageWithAttachments(ctx context.Context, messageID uuid.UUID) (*GetMessageWithAttachmentsResponse, error)
	Close() error
}

// ─── gRPC клиент ───────────────────────────────────────────────────────────

type messagesClient struct {
	client  messagesv1.MessageServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewMessagesClient(endpoint string, timeout time.Duration) (MessagesClientInterface, error) {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "grpc://")

	if endpoint == "" {
		return nil, fmt.Errorf("empty messages service endpoint")
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithReturnConnectionError(),
	}

	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to messages service: %w", err)
	}

	return &messagesClient{
		client:  messagesv1.NewMessageServiceClient(conn),
		conn:    conn,
		timeout: timeout,
	}, nil
}

func (c *messagesClient) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	replyToID := ""
	if req.ReplyToID != nil {
		replyToID = req.ReplyToID.String()
	}

	resp, err := c.client.SendMessage(callCtx, &messagesv1.SendMessageRequest{
		SenderId:    req.SenderID.String(),
		ChatId:      req.ChatID.String(),
		ServerId:    req.ServerID.String(),
		Content:     req.Content,
		ContentType: req.ContentType,
		ReplyToId:   replyToID,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC SendMessage failed: %w", err)
	}

	return &SendMessageResponse{
		MessageID: uuid.MustParse(resp.MessageId),
		CreatedAt: resp.CreatedAt.AsTime(),
	}, nil
}

func (c *messagesClient) UpdateMessageStatus(ctx context.Context, req *UpdateStatusRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.UpdateMessageStatus(callCtx, &messagesv1.UpdateMessageStatusRequest{
		MessageId: req.MessageID.String(),
		UserId:    req.UserID.String(),
		Status:    req.Status,
	})
	return err
}

func (c *messagesClient) EditMessage(ctx context.Context, req *EditMessageRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.EditMessage(callCtx, &messagesv1.EditMessageRequest{
		MessageId:  req.MessageID.String(),
		EditorId:   req.EditorID.String(),
		NewContent: req.NewContent,
	})
	return err
}

func (c *messagesClient) DeleteMessage(ctx context.Context, req *DeleteMessageRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.DeleteMessage(callCtx, &messagesv1.DeleteMessageRequest{
		MessageId: req.MessageID.String(),
		DeleterId: req.DeleterID.String(),
	})
	return err
}

func (c *messagesClient) GetMessages(ctx context.Context, req *GetMessagesRequest) (*GetMessagesResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var after *timestamppb.Timestamp
	if req.After != nil {
		after = timestamppb.New(*req.After)
	}

	resp, err := c.client.GetMessages(callCtx, &messagesv1.GetMessagesRequest{
		ChatId: req.ChatID.String(),
		Limit:  int32(req.Limit),
		Offset: int32(req.Offset),
		After:  after,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetMessages failed: %w", err)
	}

	messages := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		messages = append(messages, protoToMessage(m))
	}

	return &GetMessagesResponse{
		Messages: messages,
		HasMore:  resp.HasMore,
	}, nil
}

func (c *messagesClient) GetMessageStatuses(ctx context.Context, req *GetMessageStatusesRequest) (*GetMessageStatusesResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetMessageStatuses(callCtx, &messagesv1.GetMessageStatusesRequest{
		MessageId: req.MessageID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetMessageStatuses failed: %w", err)
	}

	statuses := make([]MessageStatus, 0, len(resp.Statuses))
	for _, s := range resp.Statuses {
		statuses = append(statuses, MessageStatus{
			UserID:    uuid.MustParse(s.UserId),
			Status:    s.Status,
			UpdatedAt: s.UpdatedAt.AsTime(),
		})
	}

	return &GetMessageStatusesResponse{Statuses: statuses}, nil
}

func (c *messagesClient) GetMessageWithAttachments(ctx context.Context, messageID uuid.UUID) (*GetMessageWithAttachmentsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	protoResp, err := c.client.GetMessageWithAttachments(callCtx, &messagesv1.GetMessageWithAttachmentsRequest{
		MessageId: messageID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetMessageWithAttachments failed: %w", err)
	}

	msg := protoToMessage(protoResp)

	attachments := make([]Attachment, 0, len(protoResp.Attachments))
	for _, a := range protoResp.Attachments {
		attachments = append(attachments, protoToAttachment(a))
	}

	return &GetMessageWithAttachmentsResponse{
		Message:     msg,
		Attachments: attachments,
	}, nil
}

func (c *messagesClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ─── Конвертеры: Proto → Domain ────────────────────────────────────────────

func protoToMessage(p *messagesv1.MessageDTO) Message {
	var replyToID *uuid.UUID
	if p.ReplyToId != "" {
		id := uuid.MustParse(p.ReplyToId)
		replyToID = &id
	}
	var serverID uuid.UUID
	if p.ServerId != "" {
		serverID = uuid.MustParse(p.ServerId)
	}

	attachments := make([]Attachment, 0, len(p.Attachments))
	for _, a := range p.Attachments {
		attachments = append(attachments, protoToAttachment(a))
	}

	return Message{
		ID:            uuid.MustParse(p.Id),
		SenderID:      uuid.MustParse(p.SenderId),
		ChatID:        uuid.MustParse(p.ChatId),
		Content:       p.Content,
		ContentType:   p.ContentType,
		HasAttachment: p.HasAttachment,
		Attachments:   attachments,
		ReplyToID:     replyToID,
		ServerID:      serverID,
		CreatedAt:     p.CreatedAt.AsTime(),
		UpdatedAt:     p.UpdatedAt.AsTime(),
		IsDeleted:     p.IsDeleted,
	}
}

func protoToAttachment(p *messagesv1.AttachmentPreview) Attachment {
	return Attachment{
		FileID:    uuid.MustParse(p.FileId),
		PublicURL: p.PublicUrl,
		MimeType:  p.MimeType,
		FileSize:  p.FileSize,
		FileName:  p.FileName,
	}
}

// ─── Конвертеры: Domain → Transport (публичные, для handler.go) ───────────

// MessageToTransportPayload — конвертирует доменное сообщение в транспортный пейлоад для WebSocket JSON
func MessageToTransportPayload(m Message) *data.MessagePayload {
	replyToID := uuidPtrToString(m.ReplyToID)
	attachments := make([]*data.AttachmentPreview, 0, len(m.Attachments))
	for _, a := range m.Attachments {
		attachments = append(attachments, AttachmentToTransportPreview(a))
	}

	return &data.MessagePayload{  // ← Возвращаем адрес структуры
		ID:            m.ID.String(),
		ChatID:        m.ChatID.String(),
		SenderID:      m.SenderID.String(),
		Content:       m.Content,
		ContentType:   m.ContentType,
		HasAttachment: m.HasAttachment,
		Attachments:   attachments,
		ReplyToID:     &replyToID,
		ServerID:      m.ServerID.String(),
		CreatedAt:     formatTime(m.CreatedAt),
		UpdatedAt:     formatTime(m.UpdatedAt),
		IsDeleted:     m.IsDeleted,
	}
}

// AttachmentToTransportPreview — конвертирует доменное вложение в транспортное
func AttachmentToTransportPreview(a Attachment) *data.AttachmentPreview {
	return &data.AttachmentPreview{
		FileID:    a.FileID.String(),
		PublicURL: a.PublicURL,
		MimeType:  a.MimeType,
		FileSize:  a.FileSize,
		FileName:  a.FileName,
	}
}

// ─── Вспомогательные функции ───────────────────────────────────────────────

func uuidPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
