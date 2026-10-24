package grpcserver_test

import (
	"context"
	"errors"
	"testing"
	"time"

	messagesv1 "github.com/MrEsbens/messenger-message-service/api/messages/v1"
	"github.com/MrEsbens/messenger-message-service/internal/domain"
	svcMocks "github.com/MrEsbens/messenger-message-service/internal/service/mocks"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcserver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ───────────────────────────────────────────────────────────
// Вспомогательные функции
// ───────────────────────────────────────────────────────────

func newTestHandler(t *testing.T) (*grpcserver.GRPCHandler, *svcMocks.MockMessageService) {
	mockSvc := svcMocks.NewMockMessageService(t)
	return grpcserver.NewGRPCHandler(mockSvc), mockSvc
}

func uuidToPtr(id uuid.UUID) *string {
	s := id.String()
	return &s
}

func tsToProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// ───────────────────────────────────────────────────────────
// Тесты SendMessage
// ───────────────────────────────────────────────────────────

func TestHandler_SendMessage_Success(t *testing.T) {
	t.Parallel()

	h, mockSvc := newTestHandler(t)

	chatID := uuid.New()
	senderID := uuid.New()
	content := "Hello, gRPC!"
	now := time.Now()

	expectedMsg := &domain.Message{
		ID:          uuid.New(),
		ChatID:      chatID,
		SenderID:    senderID,
		Content:     content,
		ContentType: domain.ContentTypeText,
		CreatedAt:   now,
	}

	mockSvc.On("SendMessage",
		mock.Anything,
		chatID,
		senderID,
		uuid.Nil,
		content,
		domain.ContentTypeText,
		(*uuid.UUID)(nil),
		false,
	).Return(expectedMsg, nil)

	req := &messagesv1.SendMessageRequest{
		SenderId:    senderID.String(),
		ChatId:      chatID.String(),
		Content:     content,
		ContentType: "text",
	}

	resp, err := h.SendMessage(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedMsg.ID.String(), resp.MessageId)
	assert.Equal(t, tsToProto(now), resp.CreatedAt)

	mockSvc.AssertExpectations(t)
}

func TestHandler_SendMessage_InvalidRequest(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandler(t)

	tests := []struct {
		name     string
		req      *messagesv1.SendMessageRequest
		wantCode codes.Code
	}{
		{
			name: "empty content",
			req: &messagesv1.SendMessageRequest{
				SenderId: uuid.New().String(),
				ChatId:   uuid.New().String(),
				Content:  "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid sender_id",
			req: &messagesv1.SendMessageRequest{
				SenderId: "not-a-uuid",
				ChatId:   uuid.New().String(),
				Content:  "test",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid chat_id",
			req: &messagesv1.SendMessageRequest{
				SenderId: uuid.New().String(),
				ChatId:   "invalid",
				Content:  "test",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid content_type",
			req: &messagesv1.SendMessageRequest{
				SenderId:    uuid.New().String(),
				ChatId:      uuid.New().String(),
				Content:     "test",
				ContentType: "html", // ❌ не в списке разрешённых
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.SendMessage(context.Background(), tt.req)
			assert.Nil(t, resp)
			assert.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestHandler_SendMessage_ServiceError(t *testing.T) {
	t.Parallel()

	h, mockSvc := newTestHandler(t)

	chatID := uuid.New()
	senderID := uuid.New()

	mockSvc.On("SendMessage",
		mock.Anything,
		chatID,
		senderID,
		uuid.Nil,
		"content",
		domain.ContentTypeText,
		(*uuid.UUID)(nil),
		false,
	).Return(nil, errors.New("database error"))

	req := &messagesv1.SendMessageRequest{
		SenderId:    senderID.String(),
		ChatId:      chatID.String(),
		Content:     "content",
		ContentType: "text",
	}

	resp, err := h.SendMessage(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))

	mockSvc.AssertExpectations(t)
}

// ───────────────────────────────────────────────────────────
// Тесты EditMessage
// ───────────────────────────────────────────────────────────

func TestHandler_EditMessage_Success(t *testing.T) {
	t.Parallel()

	h, mockSvc := newTestHandler(t)

	messageID := uuid.New()
	editorID := uuid.New()
	newContent := "Updated!"

	mockSvc.On("EditMessage",
		mock.Anything,
		messageID,
		editorID,
		newContent,
	).Return(nil)

	req := &messagesv1.EditMessageRequest{
		MessageId:  messageID.String(),
		EditorId:   editorID.String(),
		NewContent: newContent,
	}

	resp, err := h.EditMessage(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp) // google.protobuf.Empty

	mockSvc.AssertExpectations(t)
}

func TestHandler_EditMessage_PermissionDenied(t *testing.T) {
	t.Parallel()

	h, mockSvc := newTestHandler(t)

	messageID := uuid.New()
	editorID := uuid.New()

	mockSvc.On("EditMessage",
		mock.Anything,
		messageID,
		editorID,
		"new content",
	).Return(errors.New("permission denied: only sender can edit message"))

	req := &messagesv1.EditMessageRequest{
		MessageId:  messageID.String(),
		EditorId:   editorID.String(),
		NewContent: "new content",
	}

	resp, err := h.EditMessage(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Nil(t, resp)

	mockSvc.AssertExpectations(t)
}

// ───────────────────────────────────────────────────────────
// Тесты GetMessages
// ───────────────────────────────────────────────────────────

func TestHandler_GetMessages_Success(t *testing.T) {
	t.Parallel()

	h, mockSvc := newTestHandler(t)

	chatID := uuid.New()
	now := time.Now()

	// 🔧 Важно: сообщения должны быть в порядке DESC (новые первыми)
	// как их вернёт БД с ORDER BY created_at DESC
	expectedMsgs := []*domain.Message{
		{
			ID:        uuid.New(),
			ChatID:    chatID,
			SenderID:  uuid.New(),
			Content:   "Msg 2", // ← Новое сообщение ПЕРВЫМ
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        uuid.New(),
			ChatID:    chatID,
			SenderID:  uuid.New(),
			Content:   "Msg 1", // ← Старое сообщение ВТОРЫМ
			CreatedAt: now.Add(-2 * time.Hour),
		},
	}

	mockSvc.On("GetChatHistory",
		mock.Anything,
		chatID,
		10, // int
		0,  // int
	).Return(expectedMsgs, nil)

	req := &messagesv1.GetMessagesRequest{
		ChatId: chatID.String(),
		Limit:  10,
		Offset: 0,
	}

	resp, err := h.GetMessages(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Messages, 2)

	// 🔧 Теперь первое сообщение — самое новое
	assert.Equal(t, "Msg 2", resp.Messages[0].Content) // ✅
	assert.Equal(t, "Msg 1", resp.Messages[1].Content)

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetMessages_InvalidLimit(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandler(t)

	tests := []struct {
		name  string
		limit int32
	}{
		{"negative limit", -5},
		{"too large limit", 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &messagesv1.GetMessagesRequest{
				ChatId: uuid.New().String(),
				Limit:  tt.limit,
			}
			resp, err := h.GetMessages(context.Background(), req)
			assert.Nil(t, resp)
			assert.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

// ───────────────────────────────────────────────────────────
// Тесты UpdateMessageStatus
// ───────────────────────────────────────────────────────────

func TestHandler_UpdateMessageStatus_Success(t *testing.T) {
	t.Parallel()

	h, mockSvc := newTestHandler(t)

	messageID := uuid.New()
	userID := uuid.New()

	mockSvc.On("UpdateMessageStatus",
		mock.Anything,
		messageID,
		userID,
		domain.StatusRead,
	).Return(nil)

	req := &messagesv1.UpdateMessageStatusRequest{
		MessageId: messageID.String(),
		UserId:    userID.String(),
		Status:    "read",
	}

	resp, err := h.UpdateMessageStatus(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	mockSvc.AssertExpectations(t)
}

func TestHandler_UpdateMessageStatus_InvalidStatus(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandler(t)

	req := &messagesv1.UpdateMessageStatusRequest{
		MessageId: uuid.New().String(),
		UserId:    uuid.New().String(),
		Status:    "unknown_status", // ❌
	}

	resp, err := h.UpdateMessageStatus(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Nil(t, resp)
}
