package grpcserver

import (
	"context"
	"errors"

	messagesv1 "github.com/MrEsbens/messenger-message-service/api/messages/v1"
	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/repository"
	"github.com/MrEsbens/messenger-message-service/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCHandler struct {
	messagesv1.UnimplementedMessageServiceServer
	service service.MessageServiceInterface
}

func NewGRPCHandler(svc service.MessageServiceInterface) *GRPCHandler {
	return &GRPCHandler{service: svc}
}

func (h *GRPCHandler) SendMessage(
	ctx context.Context,
	req *messagesv1.SendMessageRequest,
) (*messagesv1.SendMessageResponse, error) {
	if req.ChatId == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	chatID, err := parseUUID(req.ChatId, "chat_id")
	if err != nil && req.ChatId != "" {
		return nil, err
	}

	senderID, err := parseUUID(req.SenderId, "sender_id")
	if err != nil && req.SenderId != "" {
		return nil, err
	}

	var replyToID *uuid.UUID
	if req.ReplyToId != "" {
		replyToID, err = parseUUIDPtr(req.ReplyToId, "reply_to_id")
		if err != nil {
			return nil, err
		}
	}

	var serverID uuid.UUID
	if req.ServerId != "" {
		serverID, err = parseUUID(req.ServerId, "server_id")
		if err != nil {
			return nil, err
		}
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = domain.ContentTypeText
	}
	if !domain.IsValidContentType(contentType) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid content_type: %s", contentType)
	}

	hasAttachment := req.FileId != ""

	msg, err := h.service.SendMessage(ctx, chatID, senderID, serverID, req.Content, contentType, replyToID, hasAttachment)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidInput) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}

	return &messagesv1.SendMessageResponse{
		MessageId: msg.ID.String(),
		CreatedAt: timestamppb.New(msg.CreatedAt),
	}, nil
}

func (h *GRPCHandler) GetMessages(
	ctx context.Context,
	req *messagesv1.GetMessagesRequest,
) (*messagesv1.GetMessagesResponse, error) {
	if req.ChatId == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}

	chatID, err := parseUUID(req.ChatId, "chat_id")
	if err != nil && req.ChatId != "" {
		return nil, err
	}

	const (
		minLimit  = 1
		maxLimit  = 100
		minOffset = 0
	)

	if req.Limit < minLimit || req.Limit > maxLimit {
		return nil, status.Errorf(codes.InvalidArgument,
			"limit must be between %d and %d", minLimit, maxLimit)
	}
	if req.Offset < minOffset {
		return nil, status.Errorf(codes.InvalidArgument,
			"offset must be >= %d", minOffset)
	}

	limit := int(req.Limit)
	offset := int(req.Offset)

	var msgs []*domain.Message
	if req.After != nil {
		afterTime := req.After.AsTime()
		msgs, err = h.service.GetChatHistoryAfter(ctx, chatID, afterTime, limit)
	} else {
		msgs, err = h.service.GetChatHistory(ctx, chatID, limit, offset)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get messages: %v", err)
	}

	dtoMessages := make([]*messagesv1.MessageDTO, 0, len(msgs))
	for _, msg := range msgs {
		dto := &messagesv1.MessageDTO{
			Id:            msg.ID.String(),
			SenderId:      msg.SenderID.String(),
			Content:       msg.Content,
			ContentType:   msg.ContentType,
			HasAttachment: msg.HasAttachment,
			CreatedAt:     timestamppb.New(msg.CreatedAt),
			UpdatedAt:     timestamppb.New(msg.UpdatedAt),
			IsDeleted:     msg.DeletedAt != nil,
		}

		dto.ChatId = req.ChatId
		if msg.ReplyToID != nil {
			dto.ReplyToId = msg.ReplyToID.String()
		}

		dtoMessages = append(dtoMessages, dto)
	}

	hasMore := len(msgs) == limit && limit > 0

	return &messagesv1.GetMessagesResponse{
		Messages: dtoMessages,
		HasMore:  hasMore,
	}, nil
}

func (h *GRPCHandler) GetMessageWithAttachments(
	ctx context.Context,
	req *messagesv1.GetMessageWithAttachmentsRequest,
) (*messagesv1.MessageDTO, error) {
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}

	messageID, err := parseUUID(req.MessageId, "message_id")
	if err != nil {
		return nil, err
	}

	msg, attachments, err := h.service.GetMessage(ctx, messageID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "message not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get message: %v", err)
	}

	dto := messageToDTO(msg, nil)

	for _, a := range attachments {
		dto.Attachments = append(dto.Attachments, &messagesv1.AttachmentPreview{
			FileId:    a.FileID,
			PublicUrl: a.PublicURL,
			MimeType:  a.MimeType,
			FileSize:  a.FileSize,
			FileName:  a.FileName,
		})
	}

	return dto, nil
}

func (h *GRPCHandler) EditMessage(
	ctx context.Context,
	req *messagesv1.EditMessageRequest,
) (*emptypb.Empty, error) {
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}
	if req.EditorId == "" {
		return nil, status.Error(codes.InvalidArgument, "editor_id is required")
	}
	if req.NewContent == "" {
		return nil, status.Error(codes.InvalidArgument, "new_content is required")
	}

	messageID, err := parseUUID(req.MessageId, "message_id")
	if err != nil {
		return nil, err
	}

	editorID, err := parseUUID(req.EditorId, "editor_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.EditMessage(ctx, messageID, editorID, req.NewContent); err != nil {
		if err.Error() == "permission denied: only sender can edit message" {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "message not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to edit message: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) DeleteMessage(
	ctx context.Context,
	req *messagesv1.DeleteMessageRequest,
) (*emptypb.Empty, error) {
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}
	if req.DeleterId == "" {
		return nil, status.Error(codes.InvalidArgument, "deleter_id is required")
	}

	messageID, err := parseUUID(req.MessageId, "message_id")
	if err != nil {
		return nil, err
	}

	deleterID, err := parseUUID(req.DeleterId, "deleter_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteMessage(ctx, messageID, deleterID); err != nil {
		if err.Error() == "permission denied: only sender can delete message" {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "message not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete message: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) UpdateMessageStatus(
	ctx context.Context,
	req *messagesv1.UpdateMessageStatusRequest,
) (*emptypb.Empty, error) {
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Status == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	if !domain.IsValidMessageStatus(req.Status) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status: %s (allowed: %s, %s, %s)",
			req.Status, domain.StatusSent, domain.StatusDelivered, domain.StatusRead)
	}

	messageID, err := parseUUID(req.MessageId, "message_id")
	if err != nil {
		return nil, err
	}

	userID, err := parseUUID(req.UserId, "user_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.UpdateMessageStatus(ctx, messageID, userID, req.Status); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update status: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) GetMessageStatuses(
	ctx context.Context,
	req *messagesv1.GetMessageStatusesRequest,
) (*messagesv1.GetMessageStatusesResponse, error) {
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}

	messageID, err := parseUUID(req.MessageId, "message_id")
	if err != nil {
		return nil, err
	}

	statuses, err := h.service.GetMessageStatuses(ctx, messageID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get statuses: %v", err)
	}

	dtoStatuses := make([]*messagesv1.MessageStatusDTO, 0, len(statuses))
	for _, s := range statuses {
		dto := &messagesv1.MessageStatusDTO{
			UserId:    s.UserID.String(),
			Status:    s.Status,
			UpdatedAt: timestamppb.New(s.UpdatedAt),
		}
		dtoStatuses = append(dtoStatuses, dto)
	}

	return &messagesv1.GetMessageStatusesResponse{
		Statuses: dtoStatuses,
	}, nil
}

func parseUUID(s, fieldName string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid %s: %v", fieldName, err)
	}
	return id, nil
}

func parseUUIDPtr(s, fieldName string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid %s: %v", fieldName, err)
	}
	return &id, nil
}

func messageToDTO(msg *domain.Message, chatID *string) *messagesv1.MessageDTO {
	dto := &messagesv1.MessageDTO{
		Id:            msg.ID.String(),
		SenderId:      msg.SenderID.String(),
		Content:       msg.Content,
		ContentType:   msg.ContentType,
		HasAttachment: msg.HasAttachment,
		CreatedAt:     timestamppb.New(msg.CreatedAt),
		UpdatedAt:     timestamppb.New(msg.UpdatedAt),
		IsDeleted:     msg.DeletedAt != nil,
	}

	if chatID != nil {
		dto.ChatId = *chatID
	}
	if msg.ReplyToID != nil {
		dto.ReplyToId = msg.ReplyToID.String()
	}

	return dto
}