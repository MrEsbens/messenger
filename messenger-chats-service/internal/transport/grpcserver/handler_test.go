package grpcserver_test

import (
	"context"
	"testing"

	chatsv1 "github.com/MrEsbens/messenger-chats-service/api/chats/v1"
	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/service"
	svcMocks "github.com/MrEsbens/messenger-chats-service/internal/service/mocks"
	"github.com/MrEsbens/messenger-chats-service/internal/transport/grpcserver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGRPCHandler_CreateChat_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	name := "Test Chat"
	createdBy := uuid.New()
	chatID := uuid.New()
	now := timestamppb.Now()

	mockSvc.On("CreateChat", mock.Anything, mock.Anything).Return(&service.ChatResponse{
		Chat: &domain.Chat{
			ID:        chatID,
			Type:      domain.ChatTypeGroup,
			Name:      &name,
			CreatedBy: createdBy,
			CreatedAt: now.AsTime(),
			UpdatedAt: now.AsTime(),
		},
		AvatarFileID: "",
		AvatarURL:    "",
		HasAvatar:    false,
	}, nil)

	req := &chatsv1.CreateChatRequest{
		Type:      chatsv1.ChatType_CHAT_TYPE_GROUP,
		Name:      name,
		CreatedBy: createdBy.String(),
	}

	resp, err := handler.CreateChat(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, chatID.String(), resp.ChatId)

	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_CreateChat_Fail_InvalidType(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	req := &chatsv1.CreateChatRequest{
		Type:      chatsv1.ChatType_CHAT_TYPE_UNSPECIFIED,
		CreatedBy: uuid.New().String(),
	}

	resp, err := handler.CreateChat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockSvc.AssertNotCalled(t, "CreateChat")
}

func TestGRPCHandler_GetChat_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	requesterID := uuid.New()
	chatName := "Test Chat"
	now := timestamppb.Now()

	mockSvc.On("GetChat", mock.Anything, chatID, requesterID).Return(&service.ChatResponse{
		Chat: &domain.Chat{
			ID:        chatID,
			Type:      domain.ChatTypeGroup,
			Name:      &chatName,
			CreatedBy: requesterID,
			CreatedAt: now.AsTime(),
			UpdatedAt: now.AsTime(),
		},
		AvatarFileID: "avatar-123",
		AvatarURL:    "https://example.com/avatar.png",
		HasAvatar:    true,
	}, nil)

	req := &chatsv1.GetChatRequest{
		ChatId:      chatID.String(),
		RequesterId: requesterID.String(),
	}

	resp, err := handler.GetChat(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, chatID.String(), resp.Id)
	assert.Equal(t, chatName, resp.Name)
	assert.NotNil(t, resp.AvatarFileId)
	assert.Equal(t, "avatar-123", *resp.AvatarFileId)
	assert.NotNil(t, resp.AvatarUrl)
	assert.Equal(t, "https://example.com/avatar.png", *resp.AvatarUrl)
	assert.NotNil(t, resp.HasAvatar)
	assert.True(t, *resp.HasAvatar)

	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_GetChat_Fail_NotFound(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	requesterID := uuid.New()

	mockSvc.On("GetChat", mock.Anything, chatID, requesterID).Return(nil, domain.ErrChatNotFound)

	req := &chatsv1.GetChatRequest{
		ChatId:      chatID.String(),
		RequesterId: requesterID.String(),
	}

	resp, err := handler.GetChat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_UpdateChat_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	updaterID := uuid.New()
	newName := "Updated Name"

	mockSvc.On("UpdateChat", mock.Anything, mock.Anything).Return(nil)

	req := &chatsv1.UpdateChatRequest{
		ChatId:    chatID.String(),
		UpdaterId: updaterID.String(),
		Name:      &newName,
	}

	resp, err := handler.UpdateChat(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, &emptypb.Empty{}, resp)
	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_DeleteChat_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	deleterID := uuid.New()

	mockSvc.On("DeleteChat", mock.Anything, chatID, deleterID).Return(nil)

	req := &chatsv1.DeleteChatRequest{
		ChatId:    chatID.String(),
		DeleterId: deleterID.String(),
	}

	resp, err := handler.DeleteChat(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, &emptypb.Empty{}, resp)
	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_ListUserChats_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	userID := uuid.New()
	chatID := uuid.New()
	chatName := "Test Chat"
	now := timestamppb.Now()

	mockSvc.On("ListUserChats", mock.Anything, userID, mock.Anything).Return(&service.ChatListResponse{
		Chats: []*service.ChatResponse{
			{
				Chat: &domain.Chat{
					ID:        chatID,
					Type:      domain.ChatTypeGroup,
					Name:      &chatName,
					CreatedBy: userID,
					CreatedAt: now.AsTime(),
					UpdatedAt: now.AsTime(),
				},
				AvatarFileID: "avatar-123",
				AvatarURL:    "https://example.com/avatar.png",
				HasAvatar:    true,
			},
		},
		HasMore:    false,
		NextCursor: "",
	}, nil)

	req := &chatsv1.ListUserChatsRequest{
		UserId: userID.String(),
		Limit:  50,
		Offset: 0,
	}

	resp, err := handler.ListUserChats(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Chats, 1)
	assert.Equal(t, chatID.String(), resp.Chats[0].Id)
	assert.Equal(t, chatName, resp.Chats[0].Name)
	assert.NotNil(t, resp.Chats[0].AvatarFileId)
	assert.Equal(t, "avatar-123", *resp.Chats[0].AvatarFileId)
	assert.NotNil(t, resp.Chats[0].AvatarUrl)
	assert.Equal(t, "https://example.com/avatar.png", *resp.Chats[0].AvatarUrl)
	assert.NotNil(t, resp.Chats[0].HasAvatar)
	assert.True(t, *resp.Chats[0].HasAvatar)

	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_AddMember_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	userID := uuid.New()
	addedBy := uuid.New()

	mockSvc.On("AddMember", mock.Anything, chatID, userID, addedBy, domain.RoleMember).Return(nil)

	req := &chatsv1.AddMemberRequest{
		ChatId:  chatID.String(),
		UserId:  userID.String(),
		AddedBy: addedBy.String(),
		Role:    chatsv1.MembershipRole_ROLE_MEMBER,
	}

	resp, err := handler.AddMember(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, &emptypb.Empty{}, resp)
	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_RemoveMember_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	userID := uuid.New()
	removedBy := uuid.New()

	mockSvc.On("RemoveMember", mock.Anything, chatID, userID, removedBy).Return(nil)

	req := &chatsv1.RemoveMemberRequest{
		ChatId:    chatID.String(),
		UserId:    userID.String(),
		RemovedBy: removedBy.String(),
	}

	resp, err := handler.RemoveMember(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, &emptypb.Empty{}, resp)
	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_IsMember_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	userID := uuid.New()

	mockSvc.On("IsMember", mock.Anything, chatID, userID).Return(true, domain.RoleAdmin, nil)

	req := &chatsv1.IsMemberRequest{
		ChatId: chatID.String(),
		UserId: userID.String(),
	}

	resp, err := handler.IsMember(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsMember)
	assert.Equal(t, chatsv1.MembershipRole_ROLE_ADMIN, resp.Role)

	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_CreateInvite_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	chatID := uuid.New()
	createdBy := uuid.New()
	token := "invite-token-123"

	mockSvc.On("CreateInvite", mock.Anything, chatID, createdBy, mock.Anything).Return(&domain.ChatInvite{
		Token:     token,
		ChatID:    chatID,
		CreatedBy: createdBy,
	}, nil)

	req := &chatsv1.CreateInviteRequest{
		ChatId:    chatID.String(),
		CreatedBy: createdBy.String(),
	}

	resp, err := handler.CreateInvite(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, token, resp.InviteToken)

	mockSvc.AssertExpectations(t)
}

func TestGRPCHandler_JoinByInvite_Success(t *testing.T) {
	mockSvc := svcMocks.NewMockChatService(t)
	handler := grpcserver.NewGRPCHandler(mockSvc)

	token := "invite-token-123"
	userID := uuid.New()

	mockSvc.On("JoinByInvite", mock.Anything, token, userID).Return(nil)

	req := &chatsv1.JoinByInviteRequest{
		InviteToken: token,
		UserId:      userID.String(),
	}

	resp, err := handler.JoinByInvite(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, &emptypb.Empty{}, resp)
	mockSvc.AssertExpectations(t)
}
