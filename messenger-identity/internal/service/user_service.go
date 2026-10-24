package service

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/MrEsbens/messenger-identity/internal/repository"
	"github.com/MrEsbens/messenger-identity/internal/transport/grpcclient"
	"github.com/google/uuid"
)

type UserService struct {
	identityv1.UnimplementedIdentityServiceServer

	userRepo   repository.UserRepository
	fileClient grpcclient.FileClientInterface
}

func NewUserService(userRepo repository.UserRepository, fileClient grpcclient.FileClientInterface) *UserService {
	return &UserService{
		userRepo:   userRepo,
		fileClient: fileClient,
	}
}

// ───────────────────────────────────────────────────────────
// Хелпер: конвертация доменного пользователя в DTO с аватаром
// ───────────────────────────────────────────────────────────

func toUserDTOWithAvatar(user *domain.User, avatarURL string) *identityv1.UserDTO {
	return &identityv1.UserDTO{
		Id:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		AvatarUrl: avatarURL,
		Status:    user.Status,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}

// ───────────────────────────────────────────────────────────
// Хелпер: загрузка аватара пользователя (упрощённая версия)
// Возвращает публичный URL из файла (аватары — публичные ресурсы)
// ───────────────────────────────────────────────────────────

func (s *UserService) loadUserAvatarURL(ctx context.Context, userID uuid.UUID) string {
	if s.fileClient == nil {
		return ""
	}

	fileResp, err := s.fileClient.GetFileByOwner(ctx, &filev1.GetFileByOwnerRequest{
		OwnerId:    userID.String(),
		OwnerType:  "user",
		BucketType: "avatars",
	})

	if err == nil && fileResp != nil && fileResp.PublicUrl != "" {
		return fileResp.PublicUrl
	}

	return ""
}

// ───────────────────────────────────────────────────────────
// GetMe — получение профиля текущего пользователя
// ───────────────────────────────────────────────────────────

func (s *UserService) GetMe(ctx context.Context, req *identityv1.GetMeRequest) (*identityv1.UserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	avatarURL := s.loadUserAvatarURL(ctx, userID)

	return &identityv1.UserResponse{
		User: toUserDTOWithAvatar(user, avatarURL),
	}, nil
}

// ───────────────────────────────────────────────────────────
// UpdateProfile — обновление профиля
// ───────────────────────────────────────────────────────────

func (s *UserService) UpdateProfile(ctx context.Context, req *identityv1.UpdateProfileRequest) (*identityv1.UserResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	updated := false

	if req.Username != nil && *req.Username != "" {
		user.Username = *req.Username
		updated = true
	}

	if req.Status != nil && *req.Status != "" {
		user.Status = *req.Status
		updated = true
	}

	if !updated {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	avatarURL := s.loadUserAvatarURL(ctx, userID)

	return &identityv1.UserResponse{
		User: toUserDTOWithAvatar(user, avatarURL),
	}, nil
}

// ───────────────────────────────────────────────────────────
// GetUploadAvatarURL — получение URL для загрузки аватара
// (тонкая обёртка — оставляем для явного разделения ответственности)
// ───────────────────────────────────────────────────────────

func (s *UserService) GetUploadAvatarURL(ctx context.Context, req *identityv1.GetUploadAvatarURLRequest) (*identityv1.GetUploadURLResponse, error) {
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
	}

	grpcResp, err := s.fileClient.GetUploadURL(ctx, &filev1.GetUploadURLRequest{
		OwnerId:    ownerID.String(),
		OwnerType:  "user",
		BucketType: "avatars",
		FileName:   req.FileName,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "file service error: %v", err)
	}

	return &identityv1.GetUploadURLResponse{
		FileId:    grpcResp.FileId,
		UploadUrl: grpcResp.UploadUrl,
		ObjectKey: grpcResp.ObjectKey,
		ExpiresAt: grpcResp.ExpiresAt,
	}, nil
}

// ───────────────────────────────────────────────────────────
// ConfirmAvatarUpload — подтверждение загрузки аватара
// ───────────────────────────────────────────────────────────

func (s *UserService) ConfirmAvatarUpload(ctx context.Context, req *identityv1.ConfirmUploadRequest) (*emptypb.Empty, error) {
	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	if _, err := uuid.Parse(req.FileId); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id format")
	}

	return s.fileClient.ConfirmUpload(ctx, &filev1.ConfirmUploadRequest{
		FileId: req.FileId,
	})
}

// ───────────────────────────────────────────────────────────
// SearchUsers — поиск пользователей с аватарами
// ───────────────────────────────────────────────────────────

func (s *UserService) SearchUsers(ctx context.Context, req *identityv1.SearchUsersRequest) (*identityv1.SearchUsersResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	users, err := s.userRepo.Search(ctx, req.Query, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search users: %v", err)
	}

	dtoUsers := make([]*identityv1.UserDTO, 0, len(users))
	for _, user := range users {
		avatarURL := s.loadUserAvatarURL(ctx, user.ID)
		dtoUsers = append(dtoUsers, toUserDTOWithAvatar(user, avatarURL))
	}

	return &identityv1.SearchUsersResponse{
		Users: dtoUsers,
	}, nil
}
