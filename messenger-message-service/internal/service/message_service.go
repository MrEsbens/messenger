package service

import (
	"context"
	"fmt"
	"log"
	"time"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	rediscache "github.com/MrEsbens/messenger-message-service/internal/cache/redis"
	redispub "github.com/MrEsbens/messenger-message-service/pkg/redis"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/repository"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient"
	"github.com/MrEsbens/messenger-message-service/internal/transport/rabbitmq"
	"github.com/google/uuid"
)

type AttachmentPreview struct {
	FileID    string `json:"file_id"`
	PublicURL string `json:"public_url"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	FileName  string `json:"file_name"`
}

type MessageServiceInterface interface {
	SendMessage(ctx context.Context, chatID, senderID, serverID uuid.UUID, content, contentType string, replyToID *uuid.UUID, hasAttachment bool) (*domain.Message, error)
	GetMessage(ctx context.Context, id uuid.UUID) (*domain.Message, []*AttachmentPreview, error)
	GetChatHistory(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error)
	GetChatHistoryWithAttachments(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, map[uuid.UUID][]*AttachmentPreview, error)
	GetChatHistoryAfter(ctx context.Context, chatID uuid.UUID, afterTime time.Time, limit int) ([]*domain.Message, error)
	EditMessage(ctx context.Context, messageID, editorID uuid.UUID, newContent string) error
	DeleteMessage(ctx context.Context, messageID, deleterID uuid.UUID) error
	UpdateMessageStatus(ctx context.Context, messageID, userID uuid.UUID, newStatus string) error
	GetMessageStatuses(ctx context.Context, messageID uuid.UUID) ([]*repository.MessageStatus, error)
	MarkChatAsRead(ctx context.Context, chatID, userID uuid.UUID) error
}

var _ MessageServiceInterface = (*MessageService)(nil)

type MessageService struct {
	msgRepo    repository.MessageRepository
	statusRepo repository.StatusRepository

	redis       redispub.RedisClientInterface
	redisPrefix string

	fileClient       grpcclient.FileClientInterface
	chatClient       grpcclient.ChatClientInterface
	serversClient    grpcclient.ServersClientInterface
	moderationClient grpcclient.ModerationClientInterface

	moderationProducer rabbitmq.ModerationProducerInterface

	modConfigCache rediscache.ModerationConfigCacheInterface
}

func NewMessageService(
	msgRepo repository.MessageRepository,
	statusRepo repository.StatusRepository,
	redisClient redispub.RedisClientInterface,
	fileClient grpcclient.FileClientInterface,
	chatClient grpcclient.ChatClientInterface,
	serversClient grpcclient.ServersClientInterface,
	moderationClient grpcclient.ModerationClientInterface,
	moderationProducer rabbitmq.ModerationProducerInterface,
	modConfigCache rediscache.ModerationConfigCacheInterface,
) MessageServiceInterface {
	return &MessageService{
		msgRepo:            msgRepo,
		statusRepo:         statusRepo,
		redis:              redisClient,
		redisPrefix:        "msg:",
		fileClient:         fileClient,
		chatClient:         chatClient,
		serversClient:      serversClient,
		moderationClient:   moderationClient,
		moderationProducer: moderationProducer,
		modConfigCache:     modConfigCache,
	}
}

func (s *MessageService) SendMessage(
	ctx context.Context,
	chatID, senderID, serverID uuid.UUID,
	content, contentType string,
	replyToID *uuid.UUID,
	hasAttachment bool,
) (*domain.Message, error) {
	// ─── 1. Валидация ───────────────────────────────────────
	if !domain.IsValidContentType(contentType) {
		log.Printf("❌ [SendMessage] Invalid content type: %s", contentType)
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}
	if content == "" {
		log.Printf("❌ [SendMessage] Empty content")
		return nil, repository.ErrInvalidInput
	}

	// ─── 2. Проверка прав (членство в чате/сервере) ─────────
	if serverID != uuid.Nil {
		isMember, err := s.serversClient.IsMember(ctx, serverID.String(), senderID.String())
		if err != nil {
			log.Printf("❌ [SendMessage] Failed to check server membership: %v", err)
			return nil, fmt.Errorf("failed to check server membership: %w", err)
		}
		if !isMember {
			log.Printf("❌ [SendMessage] Permission denied: user %s not member of server %s",
				senderID, serverID)
			return nil, domain.ErrPermissionDenied
		}
	} else if s.chatClient != nil {
		result, err := s.chatClient.IsMember(ctx, chatID.String(), senderID.String())
		if err != nil {
			log.Printf("❌ [SendMessage] Failed to check chat membership: %v", err)
			return nil, fmt.Errorf("failed to check membership: %w", err)
		}
		if !result.IsMember {
			log.Printf("❌ [SendMessage] Permission denied: user %s not member of chat %s",
				senderID, chatID)
			return nil, domain.ErrPermissionDenied
		}
	}

	// ─── 3. Создаём сообщение в БД ──────────────────────────
	msg := &domain.Message{
		ChatID:        chatID,
		SenderID:      senderID,
		Content:       content,
		ContentType:   contentType,
		ReplyToID:     replyToID,
		HasAttachment: hasAttachment,
	}

	if err := s.msgRepo.Create(ctx, msg); err != nil {
		log.Printf("❌ [SendMessage] Failed to create message in DB: %v", err)
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// ─── 4. Ставим статус "sent" ────────────────────────────
	_ = s.statusRepo.Upsert(ctx, &repository.MessageStatus{
		MessageID: msg.ID,
		UserID:    senderID,
		Status:    domain.StatusSent,
	})

	// ─── 5. Публикуем в Redis (мгновенная доставка) ─────────
	attachments := s.getAttachmentPreviews(ctx, msg.ID)
	redisChannel := s.redisChannel("new_message", chatID.String())
	redisEvent := NewMessageEvent{
		Type:        "new_message",
		ChatID:      chatID.String(),
		Message:     s.messageToPayload(msg, attachments),
		Attachments: attachments,
		PublishedAt: time.Now(),
	}
	if err := s.redis.Publish(ctx, redisChannel, redisEvent); err != nil {
		log.Printf("⚠️ [SendMessage] Failed to publish to Redis: %v", err)
	}

	// ─── 6. Асинхронно публикуем в RabbitMQ для модерации ───
	if s.moderationProducer != nil && serverID != uuid.Nil {
		modCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		go func() {
			defer cancel()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("❌ [SendMessage] Panic in moderation publish goroutine: %v", r)
				}
			}()

			req := domain.NewModerationRequest(msg, serverID)

			if err := s.moderationProducer.Publish(modCtx, req); err != nil {
				log.Printf("❌ [SendMessage] RabbitMQ Publish FAILED for message %s: %v", msg.ID, err)
			} else {
				log.Printf("✅ [SendMessage] RabbitMQ Publish SUCCESS for message %s to exchange %s",
					msg.ID, "moderation.events")
			}
		}()
	} else {
		if s.moderationProducer == nil {
			log.Printf("⚠️ [SendMessage] SKIPPED: moderationProducer is nil")
		}
		if serverID == uuid.Nil {
			log.Printf("⚠️ [SendMessage] SKIPPED: serverID is Nil (private/group chat, no moderation)")
		}
	}

	// ─── 7. Возвращаем сообщение сразу ──────────────────────
	return msg, nil
}

func (s *MessageService) getAttachmentPreviews(ctx context.Context, messageID uuid.UUID) []*AttachmentPreview {
	if s.fileClient == nil {
		return nil
	}

	attachCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := s.fileClient.GetFilesByOwner(attachCtx, &filev1.GetFilesByOwnerRequest{
		OwnerId:    messageID.String(),
		OwnerType:  domain.FileOwnerTypeMessage,
		BucketType: domain.FileBucketAttachments,
	})

	if err != nil {
		return nil
	}

	preview := make([]*AttachmentPreview, 0, len(resp.Files))
	for _, f := range resp.Files {
		preview = append(preview, &AttachmentPreview{
			FileID:    f.FileId,
			PublicURL: f.PublicUrl,
			MimeType:  f.MimeType,
			FileSize:  f.FileSize,
			FileName:  f.FileName,
		})
	}

	return preview
}

func (s *MessageService) messageToPayload(msg *domain.Message, attachments []*AttachmentPreview) *MessagePayload {
	return &MessagePayload{
		ID:            msg.ID.String(),
		ChatID:        msg.ChatID.String(),
		SenderID:      msg.SenderID.String(),
		Content:       msg.Content,
		ContentType:   msg.ContentType,
		HasAttachment: msg.HasAttachment,
		Attachments:   attachments,
		ReplyToID:     uuidPtrToString(msg.ReplyToID),
		CreatedAt:     msg.CreatedAt,
		UpdatedAt:     msg.UpdatedAt,
		IsDeleted:     msg.DeletedAt != nil,
	}
}

func (s *MessageService) GetMessage(ctx context.Context, id uuid.UUID) (*domain.Message, []*AttachmentPreview, error) {
	msg, err := s.msgRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get message: %w", err)
	}

	attachments := s.getAttachmentPreviews(ctx, id)

	return msg, attachments, nil
}

func (s *MessageService) GetChatHistory(
	ctx context.Context,
	chatID uuid.UUID,
	limit, offset int,
) ([]*domain.Message, error) {
	msgs, err := s.msgRepo.GetByChat(ctx, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}
	return msgs, nil
}

func (s *MessageService) GetChatHistoryWithAttachments(
	ctx context.Context,
	chatID uuid.UUID,
	limit, offset int,
) ([]*domain.Message, map[uuid.UUID][]*AttachmentPreview, error) {
	msgs, err := s.msgRepo.GetByChat(ctx, chatID, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	attachmentsMap := make(map[uuid.UUID][]*AttachmentPreview)

	for _, msg := range msgs {
		if msg.HasAttachment {
			attachCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			attachmentsMap[msg.ID] = s.getAttachmentPreviews(attachCtx, msg.ID)
			cancel()
		}
	}

	return msgs, attachmentsMap, nil
}

func (s *MessageService) GetChatHistoryAfter(
	ctx context.Context,
	chatID uuid.UUID,
	afterTime time.Time,
	limit int,
) ([]*domain.Message, error) {
	msgs, err := s.msgRepo.GetByChatAfter(ctx, chatID, afterTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history after: %w", err)
	}
	return msgs, nil
}

func (s *MessageService) EditMessage(
	ctx context.Context,
	messageID, editorID uuid.UUID,
	newContent string,
) error {
	msg, err := s.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if msg.SenderID != editorID {
		return fmt.Errorf("permission denied: only sender can edit message")
	}

	if s.chatClient != nil {
		result, err := s.chatClient.IsMember(ctx, msg.ChatID.String(), editorID.String())
		if err != nil {
			return fmt.Errorf("failed to check membership: %w", err)
		}
		if !result.IsMember {
			return domain.ErrPermissionDenied
		}
	}

	if err := s.msgRepo.Update(ctx, messageID, newContent); err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	attachments := s.getAttachmentPreviews(ctx, messageID)

	channel := s.redisChannel("message_updated", msg.ChatID.String())
	event := MessageUpdatedEvent{
		Type:        "message_updated",
		ChatID:      msg.ChatID.String(),
		MessageID:   messageID.String(),
		NewContent:  newContent,
		UpdatedAt:   time.Now(),
		EditorID:    editorID.String(),
		Attachments: attachments,
	}
	_ = s.redis.Publish(ctx, channel, event)

	return nil
}

func (s *MessageService) DeleteMessage(
	ctx context.Context,
	messageID, deleterID uuid.UUID,
) error {
	msg, err := s.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if deleterID != domain.SystemModeratorID {
		if msg.SenderID != deleterID {
			return fmt.Errorf("permission denied: only sender or system moderator can delete message")
		}

		if s.chatClient != nil {
			result, err := s.chatClient.IsMember(ctx, msg.ChatID.String(), deleterID.String())
			if err != nil {
				return fmt.Errorf("failed to check membership: %w", err)
			}
			if !result.IsMember {
				return domain.ErrPermissionDenied
			}
		}
	}

	if err := s.msgRepo.Delete(ctx, messageID); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	log.Printf("🗑️ [DeleteMessage] Message %s deleted by %s, event published to Redis channel %s",
		messageID, deleterID)

	attachments := s.getAttachmentPreviews(ctx, messageID)

	channel := s.redisChannel("message_deleted", msg.ChatID.String())
	event := MessageDeletedEvent{
		Type:        "message_deleted",
		ChatID:      msg.ChatID.String(),
		MessageID:   messageID.String(),
		DeletedAt:   time.Now(),
		DeleterID:   deleterID.String(),
		Attachments: attachments,
	}
	_ = s.redis.Publish(ctx, channel, event)

	return nil
}

func (s *MessageService) UpdateMessageStatus(
	ctx context.Context,
	messageID, userID uuid.UUID,
	newStatus string,
) error {
	if !domain.IsValidMessageStatus(newStatus) {
		return fmt.Errorf("invalid status: %s", newStatus)
	}

	current, err := s.statusRepo.GetByUserAndMessage(ctx, userID, messageID)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("failed to get current status: %w", err)
	}

	if current != nil {
		if !domain.CanTransitionTo(current.Status, newStatus) {
			return nil
		}
	}

	status := &repository.MessageStatus{
		MessageID: messageID,
		UserID:    userID,
		Status:    newStatus,
	}
	if err := s.statusRepo.Upsert(ctx, status); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	msg, err := s.msgRepo.GetByID(ctx, messageID)
	if err == nil {
		channel := s.redisChannel("status_updated", msg.ChatID.String())
		event := StatusUpdatedEvent{
			Type:      "status_updated",
			ChatID:    msg.ChatID.String(),
			MessageID: messageID.String(),
			UserID:    userID.String(),
			Status:    newStatus,
			UpdatedAt: time.Now(),
		}
		_ = s.redis.Publish(ctx, channel, event)
	}

	return nil
}

func (s *MessageService) GetMessageStatuses(
	ctx context.Context,
	messageID uuid.UUID,
) ([]*repository.MessageStatus, error) {
	statuses, err := s.statusRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message statuses: %w", err)
	}
	return statuses, nil
}

func (s *MessageService) MarkChatAsRead(
	ctx context.Context,
	chatID, userID uuid.UUID,
) error {
	msgs, err := s.msgRepo.GetByChat(ctx, chatID, 100, 0)
	if err != nil {
		return fmt.Errorf("failed to get chat messages: %w", err)
	}

	updated := 0
	for _, msg := range msgs {
		if msg.SenderID == userID {
			continue
		}

		_ = s.UpdateMessageStatus(ctx, msg.ID, userID, domain.StatusRead)
		updated++
	}

	return nil
}

func (s *MessageService) redisChannel(eventType, targetID string) string {
	return fmt.Sprintf("%s%s:%s", s.redisPrefix, eventType, targetID)
}

func uuidPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// ───────────────────────────────────────────────────────────
// События для Redis Pub/Sub (без изменений)
// ───────────────────────────────────────────────────────────

// MessagePayload — данные сообщения для отправки через WebSocket/Redis
type MessagePayload struct {
	ID            string               `json:"id"`
	ChatID        string               `json:"chat_id"`
	SenderID      string               `json:"sender_id"`
	Content       string               `json:"content"`
	ContentType   string               `json:"content_type"`
	HasAttachment bool                 `json:"has_attachment"`
	Attachments   []*AttachmentPreview `json:"attachments,omitempty"`
	ReplyToID     string               `json:"reply_to_id,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	IsDeleted     bool                 `json:"is_deleted"`
}

// NewMessageEvent — событие нового сообщения
type NewMessageEvent struct {
	Type        string               `json:"type"`
	ChatID      string               `json:"chat_id"`
	Message     *MessagePayload      `json:"message"`
	Attachments []*AttachmentPreview `json:"attachments,omitempty"`
	PublishedAt time.Time            `json:"published_at"`
}

// MessageUpdatedEvent — событие обновления сообщения
type MessageUpdatedEvent struct {
	Type        string               `json:"type"`
	ChatID      string               `json:"chat_id"`
	MessageID   string               `json:"message_id"`
	NewContent  string               `json:"new_content"`
	UpdatedAt   time.Time            `json:"updated_at"`
	EditorID    string               `json:"editor_id"`
	Attachments []*AttachmentPreview `json:"attachments,omitempty"`
}

// MessageDeletedEvent — событие удаления сообщения
type MessageDeletedEvent struct {
	Type        string               `json:"type"`
	ChatID      string               `json:"chat_id"`
	MessageID   string               `json:"message_id"`
	DeletedAt   time.Time            `json:"deleted_at"`
	DeleterID   string               `json:"deleter_id"`
	Attachments []*AttachmentPreview `json:"attachments,omitempty"`
}

// StatusUpdatedEvent — событие обновления статуса
type StatusUpdatedEvent struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}
