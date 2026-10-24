package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/MrEsbens/messenger-identity/internal/domain"
	repoMocks "github.com/MrEsbens/messenger-identity/internal/repository/mocks"
	"github.com/MrEsbens/messenger-identity/internal/service"
	clientMocks "github.com/MrEsbens/messenger-identity/internal/transport/grpcclient/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newTestUserService(t *testing.T) (*service.UserService, *repoMocks.MockUserRepository, *clientMocks.MockFileClient) {
	userRepo := repoMocks.NewMockUserRepository(t)
	fileClient := clientMocks.NewMockFileClient(t)
	return service.NewUserService(userRepo, fileClient), userRepo, fileClient
}

func createTestUser() *domain.User {
	return &domain.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		Status:    "online",
		CreatedAt: time.Now(),
	}
}

func TestUserService_GetMe_WithAvatar(t *testing.T) {
	svc, userRepo, fileClient := newTestUserService(t)

	user := createTestUser()
	req := &identityv1.GetMeRequest{UserId: user.ID.String()}

	userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil).Once()

	fileClient.On("GetFileByOwner", mock.Anything, &filev1.GetFileByOwnerRequest{
		OwnerId:    user.ID.String(),
		OwnerType:  "user",
		BucketType: "avatars",
	}).Return(&filev1.GetFileByOwnerResponse{
		FileId:    uuid.New().String(),
		ObjectKey: "avatars/users/" + user.ID.String() + "/avatar.png",
		PublicUrl: "http://localhost:9000/avatars/users/" + user.ID.String() + "/avatar.png",
		MimeType:  "image/png",
		FileSize:  1024,
	}, nil).Once()

	resp, err := svc.GetMe(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, user.Username, resp.User.Username)
	assert.Contains(t, resp.User.AvatarUrl, "/avatars/users/"+user.ID.String()+"/")

	userRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}

func TestUserService_GetMe_NoAvatar(t *testing.T) {
	svc, userRepo, fileClient := newTestUserService(t)

	user := createTestUser()
	req := &identityv1.GetMeRequest{UserId: user.ID.String()}

	userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil).Once()

	fileClient.On("GetFileByOwner", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "no avatar")).Once()

	resp, err := svc.GetMe(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "", resp.User.AvatarUrl)

	userRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}

func TestUserService_GetMe_InvalidUserID(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	req := &identityv1.GetMeRequest{UserId: "invalid-uuid"}

	resp, err := svc.GetMe(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUserService_GetMe_UserNotFound(t *testing.T) {
	svc, userRepo, _ := newTestUserService(t)

	userID := uuid.New()
	req := &identityv1.GetMeRequest{UserId: userID.String()}

	userRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("not found")).Once()

	resp, err := svc.GetMe(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	userRepo.AssertExpectations(t)
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	svc, userRepo, fileClient := newTestUserService(t)

	user := createTestUser()
	newUsername := "updated_username"
	req := &identityv1.UpdateProfileRequest{
		UserId:   user.ID.String(),
		Username: &newUsername,
	}

	userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil).Once()
	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.Username == newUsername
	})).Return(nil).Once()

	fileClient.On("GetFileByOwner", mock.Anything, &filev1.GetFileByOwnerRequest{
		OwnerId:    user.ID.String(),
		OwnerType:  "user",
		BucketType: "avatars",
	}).Return(&filev1.GetFileByOwnerResponse{
		PublicUrl: "http://localhost:9000/avatars/users/" + user.ID.String() + "/avatar.png",
	}, nil).Once()

	resp, err := svc.UpdateProfile(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, newUsername, resp.User.Username)
	assert.Contains(t, resp.User.AvatarUrl, "/avatars/users/")

	userRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}

func TestUserService_UpdateProfile_UpdateStatus(t *testing.T) {
	svc, userRepo, fileClient := newTestUserService(t)

	user := createTestUser()
	newStatus := "away"
	req := &identityv1.UpdateProfileRequest{
		UserId: user.ID.String(),
		Status: &newStatus,
	}

	userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil).Once()
	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.Status == newStatus
	})).Return(nil).Once()

	fileClient.On("GetFileByOwner", mock.Anything, &filev1.GetFileByOwnerRequest{
		OwnerId:    user.ID.String(),
		OwnerType:  "user",
		BucketType: "avatars",
	}).Return(&filev1.GetFileByOwnerResponse{
		PublicUrl: "http://localhost:9000/avatars/users/" + user.ID.String() + "/avatar.png",
	}, nil).Once()

	resp, err := svc.UpdateProfile(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, newStatus, resp.User.Status)
	assert.Contains(t, resp.User.AvatarUrl, "/avatars/users/")

	userRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}

func TestUserService_UpdateProfile_NoFieldsToUpdate(t *testing.T) {
	svc, userRepo, _ := newTestUserService(t)

	user := createTestUser()
	emptyUsername := ""
	emptyStatus := ""
	req := &identityv1.UpdateProfileRequest{
		UserId:   user.ID.String(),
		Username: &emptyUsername,
		Status:   &emptyStatus,
	}

	userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil).Once()

	resp, err := svc.UpdateProfile(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	userRepo.AssertExpectations(t)
}

func TestUserService_UpdateProfile_UserNotFound(t *testing.T) {
	svc, userRepo, _ := newTestUserService(t)

	userID := uuid.New()
	newUsername := "updated"
	req := &identityv1.UpdateProfileRequest{
		UserId:   userID.String(),
		Username: &newUsername,
	}

	userRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("not found")).Once()

	resp, err := svc.UpdateProfile(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	userRepo.AssertExpectations(t)
}

func TestUserService_GetUploadAvatarURL_Success(t *testing.T) {
	svc, _, fileClient := newTestUserService(t)

	userID := uuid.New()
	req := &identityv1.GetUploadAvatarURLRequest{
		OwnerId:  userID.String(),
		FileName: "avatar.png",
		MimeType: "image/png",
		FileSize: 102400,
	}

	grpcResp := &filev1.GetUploadURLResponse{
		FileId:    uuid.New().String(),
		UploadUrl: "http://localhost:9000/avatars/users/.../file.png?sig=abc",
		ObjectKey: "avatars/users/" + userID.String() + "/file.png",
		ExpiresAt: timestamppb.New(time.Now().Add(15 * time.Minute)),
	}

	fileClient.On("GetUploadURL", mock.Anything, &filev1.GetUploadURLRequest{
		OwnerId:    userID.String(),
		OwnerType:  "user",
		BucketType: "avatars",
		FileName:   "avatar.png",
		MimeType:   "image/png",
		FileSize:   102400,
	}).Return(grpcResp, nil).Once()

	resp, err := svc.GetUploadAvatarURL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, grpcResp.FileId, resp.FileId)
	assert.Equal(t, grpcResp.UploadUrl, resp.UploadUrl)

	fileClient.AssertExpectations(t)
}

func TestUserService_GetUploadAvatarURL_InvalidOwnerID(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	req := &identityv1.GetUploadAvatarURLRequest{
		OwnerId: "invalid-uuid",
	}

	resp, err := svc.GetUploadAvatarURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUserService_ConfirmAvatarUpload_Success(t *testing.T) {
	svc, _, fileClient := newTestUserService(t)

	fileID := uuid.New().String()
	req := &identityv1.ConfirmUploadRequest{FileId: fileID}

	fileClient.On("ConfirmUpload", mock.Anything, &filev1.ConfirmUploadRequest{
		FileId: fileID,
	}).Return(&emptypb.Empty{}, nil).Once()

	resp, err := svc.ConfirmAvatarUpload(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	fileClient.AssertExpectations(t)
}

func TestUserService_ConfirmAvatarUpload_InvalidFileID(t *testing.T) {
	svc, _, fileClient := newTestUserService(t)

	req := &identityv1.ConfirmUploadRequest{FileId: "invalid"}

	resp, err := svc.ConfirmAvatarUpload(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	fileClient.AssertNotCalled(t, "ConfirmUpload")
}

func TestUserService_SearchUsers_Success(t *testing.T) {
	svc, userRepo, fileClient := newTestUserService(t)

	users := []*domain.User{
		{ID: uuid.New(), Username: "alice", Email: "alice@example.com", Status: "online", CreatedAt: time.Now()},
		{ID: uuid.New(), Username: "bob", Email: "bob@example.com", Status: "away", CreatedAt: time.Now()},
	}

	req := &identityv1.SearchUsersRequest{
		Query:  "ali",
		Limit:  10,
		Offset: 0,
	}

	userRepo.On("Search", mock.Anything, "ali", 10, 0).Return(users, nil).Once()

	for _, u := range users {
		fileClient.On("GetFileByOwner", mock.Anything, &filev1.GetFileByOwnerRequest{
			OwnerId:    u.ID.String(),
			OwnerType:  "user",
			BucketType: "avatars",
		}).Return(&filev1.GetFileByOwnerResponse{
			FileId:    uuid.New().String(),
			PublicUrl: "http://localhost:9000/avatars/users/" + u.ID.String() + "/avatar.png",
		}, nil).Once()
	}

	resp, err := svc.SearchUsers(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, "alice", resp.Users[0].Username)
	assert.Contains(t, resp.Users[0].AvatarUrl, "/avatars/users/")

	userRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}

func TestUserService_SearchUsers_NoAvatars(t *testing.T) {
	svc, userRepo, fileClient := newTestUserService(t)

	users := []*domain.User{
		{ID: uuid.New(), Username: "charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	req := &identityv1.SearchUsersRequest{Query: "charlie"}

	userRepo.On("Search", mock.Anything, "charlie", 20, 0).Return(users, nil).Once()

	fileClient.On("GetFileByOwner", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "no avatar")).Once()

	resp, err := svc.SearchUsers(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 1)
	assert.Equal(t, "", resp.Users[0].AvatarUrl)

	userRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}

func TestUserService_SearchUsers_EmptyQuery(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	req := &identityv1.SearchUsersRequest{Query: ""}

	resp, err := svc.SearchUsers(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUserService_SearchUsers_LimitClamped(t *testing.T) {
	svc, userRepo, fileClient := newTestUserService(t)

	req := &identityv1.SearchUsersRequest{
		Query:  "test",
		Limit:  200, // > 100, должно обрезаться
		Offset: 10,
	}

	userRepo.On("Search", mock.Anything, "test", 100, 10).Return([]*domain.User{}, nil).Once()

	resp, err := svc.SearchUsers(context.Background(), req)

	require.NoError(t, err)
	assert.Empty(t, resp.Users)

	userRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}
