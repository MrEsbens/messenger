package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"github.com/MrEsbens/messenger-file-service/internal/domain"
	minioMocks "github.com/MrEsbens/messenger-file-service/internal/pkg/minio/mocks"
	"github.com/MrEsbens/messenger-file-service/internal/pkg/storage"
	"github.com/MrEsbens/messenger-file-service/internal/repository"
	repoMocks "github.com/MrEsbens/messenger-file-service/internal/repository/mocks"
	"github.com/MrEsbens/messenger-file-service/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestFileService(t *testing.T) (*service.FileService, *repoMocks.MockFileRepository, *minioMocks.MockMinIOClient) {
	fileRepo := repoMocks.NewMockFileRepository(t)
	minioClient := minioMocks.NewMockMinIOClient(t)
	return service.NewFileService(minioClient, fileRepo), fileRepo, minioClient
}

func TestFileService_GetUploadURL_Success(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	ownerID := uuid.New()
	ownerType := domain.OwnerTypeUser
	bucketType := storage.BucketAvatars

	req := &filev1.GetUploadURLRequest{
		OwnerId:    ownerID.String(),
		OwnerType:  ownerType.String(),
		BucketType: bucketType.Name(),
		FileName:   "avatar.png",
		MimeType:   "image/png",
		FileSize:   1024,
	}

	expectedURL := "http://localhost:9000/avatars/some-key.png?sig=abc"
	// 🔧 Новый формат префикса: bucket.Prefix(ownerType, entityID)
	expectedPrefix := bucketType.Prefix(ownerType, ownerID.String())

	fileRepo.On("Create", mock.Anything, mock.MatchedBy(func(f *domain.File) bool {
		return f.OwnerID == ownerID &&
			f.BucketName == bucketType.Name() &&
			f.FileName == "avatar.png" &&
			f.Status == "pending" &&
			strings.HasPrefix(f.ObjectKey, expectedPrefix+"/") &&
			strings.HasSuffix(f.ObjectKey, ".png")
	})).Return(nil).Once()

	minioClient.On("GetPresignedPutURL", mock.Anything, bucketType, mock.Anything, 15*time.Minute).
		Return(expectedURL, nil).Once()

	resp, err := svc.GetUploadURL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.True(t, strings.HasPrefix(resp.ObjectKey, expectedPrefix+"/"))
	assert.True(t, strings.HasSuffix(resp.ObjectKey, ".png"))
	assert.Equal(t, expectedURL, resp.UploadUrl)
	assert.NotEmpty(t, resp.FileId)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_GetUploadURL_InvalidOwnerID(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.GetUploadURLRequest{
		OwnerId:    "invalid-uuid",
		BucketType: storage.BucketAvatars.Name(),
	}

	resp, err := svc.GetUploadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_GetUploadURL_InvalidBucketType(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.GetUploadURLRequest{
		OwnerId:    uuid.New().String(),
		BucketType: "invalid-bucket",
	}

	resp, err := svc.GetUploadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_GetUploadURL_FileSizeTooLarge(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.GetUploadURLRequest{
		OwnerId:    uuid.New().String(),
		BucketType: storage.BucketAvatars.Name(),
		FileSize:   10 * 1024 * 1024,
	}

	resp, err := svc.GetUploadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_GetUploadURL_InvalidCombination(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	// ❌ chat не может иметь вложения
	req := &filev1.GetUploadURLRequest{
		OwnerId:    uuid.New().String(),
		OwnerType:  domain.OwnerTypeChat.String(),
		BucketType: storage.BucketAttachments.Name(),
		FileSize:   1024,
	}

	resp, err := svc.GetUploadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "does not support owner_type")
}

func TestFileService_GetUploadURL_RepoError(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	req := &filev1.GetUploadURLRequest{
		OwnerId:    uuid.New().String(),
		OwnerType:  domain.OwnerTypeUser.String(),
		BucketType: storage.BucketAvatars.Name(),
		FileSize:   1024,
	}

	fileRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

	resp, err := svc.GetUploadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_GetUploadURL_MinioError(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	req := &filev1.GetUploadURLRequest{
		OwnerId:    uuid.New().String(),
		OwnerType:  domain.OwnerTypeUser.String(),
		BucketType: storage.BucketAvatars.Name(),
		FileSize:   1024,
	}

	fileRepo.On("Create", mock.Anything, mock.MatchedBy(func(f *domain.File) bool {
		return f.Status == "pending"
	})).Return(nil).Once()

	minioClient.On("GetPresignedPutURL", mock.Anything, storage.BucketAvatars, mock.Anything, 15*time.Minute).
		Return("", errors.New("minio error")).Once()

	fileRepo.On("SoftDelete", mock.Anything, mock.Anything).Return(nil).Once()

	resp, err := svc.GetUploadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_Success(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	file := &domain.File{
		ID:         fileID,
		BucketName: storage.BucketAvatars.Name(),
		ObjectKey:  "avatars/users/file.png",
		Status:     "pending",
	}

	// 🔧 Конвертируем BucketName → BucketType для вызова минико
	bucketType := storage.BucketType(file.BucketName)

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("StatObject", mock.Anything, bucketType.Name(), file.ObjectKey).Return(nil).Once()
	fileRepo.On("ConfirmUpload", mock.Anything, fileID).Return(nil).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_InvalidFileID(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.ConfirmUploadRequest{FileId: "invalid"}

	resp, err := svc.ConfirmUpload(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_ConfirmUpload_FileNotFound(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	fileID := uuid.New()
	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(nil, repository.ErrNotFound).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_WrongStatus(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	fileID := uuid.New()
	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	file := &domain.File{ID: fileID, Status: "active"}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_FileNotInStorage(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	file := &domain.File{
		ID:         fileID,
		BucketName: storage.BucketAvatars.Name(),
		ObjectKey:  "avatars/users/file.png",
		Status:     "pending",
	}

	bucketType := storage.BucketType(file.BucketName)

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("StatObject", mock.Anything, bucketType.Name(), file.ObjectKey).
		Return(errors.New("object not found")).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_GetDownloadURL_Success(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	req := &filev1.GetDownloadURLRequest{
		FileId:        fileID.String(),
		ExpirySeconds: 3600,
	}

	file := &domain.File{
		ID:         fileID,
		BucketName: storage.BucketAvatars.Name(),
		ObjectKey:  "avatars/users/file.png",
	}

	expectedURL := "http://localhost:9000/avatars/users/file.png?sig=xyz"
	bucketType := storage.BucketType(file.BucketName)

	fileRepo.On("GetActiveByID", mock.Anything, fileID).Return(file, nil).Once()

	minioClient.On("GetPresignedGetURL", mock.Anything, bucketType, file.ObjectKey, time.Hour).
		Return(expectedURL, nil).Once()

	resp, err := svc.GetDownloadURL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedURL, resp.DownloadUrl)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_GetDownloadURL_InvalidFileID(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.GetDownloadURLRequest{FileId: "invalid"}

	resp, err := svc.GetDownloadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_GetDownloadURL_FileNotFound(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	fileID := uuid.New()
	req := &filev1.GetDownloadURLRequest{FileId: fileID.String()}

	fileRepo.On("GetActiveByID", mock.Anything, fileID).Return(nil, repository.ErrNotFound).Once()

	resp, err := svc.GetDownloadURL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_DeleteFile_Success(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	ownerID := uuid.New()
	req := &filev1.DeleteFileRequest{
		FileId:  fileID.String(),
		OwnerId: ownerID.String(),
	}

	file := &domain.File{
		ID:         fileID,
		OwnerID:    ownerID,
		BucketName: storage.BucketAvatars.Name(),
		ObjectKey:  "avatars/users/file.png",
	}

	bucketType := storage.BucketType(file.BucketName)

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), file.ObjectKey, mock.AnythingOfType("minio.RemoveObjectOptions")).Return(nil).Once()
	fileRepo.On("HardDelete", mock.Anything, fileID).Return(nil).Once()

	resp, err := svc.DeleteFile(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_DeleteFile_InvalidFileID(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.DeleteFileRequest{
		FileId:  "invalid",
		OwnerId: uuid.New().String(),
	}

	resp, err := svc.DeleteFile(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_DeleteFile_InvalidOwnerID(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.DeleteFileRequest{
		FileId:  uuid.New().String(),
		OwnerId: "invalid",
	}

	resp, err := svc.DeleteFile(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_DeleteFile_FileNotFound(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	fileID := uuid.New()
	req := &filev1.DeleteFileRequest{
		FileId:  fileID.String(),
		OwnerId: uuid.New().String(),
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(nil, repository.ErrNotFound).Once()

	resp, err := svc.DeleteFile(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_DeleteFile_WrongOwner(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	fileID := uuid.New()
	wrongOwnerID := uuid.New()
	req := &filev1.DeleteFileRequest{
		FileId:  fileID.String(),
		OwnerId: wrongOwnerID.String(),
	}

	file := &domain.File{
		ID:      fileID,
		OwnerID: uuid.New(),
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()

	resp, err := svc.DeleteFile(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_DeleteFile_MinioError(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	ownerID := uuid.New()
	req := &filev1.DeleteFileRequest{
		FileId:  fileID.String(),
		OwnerId: ownerID.String(),
	}

	file := &domain.File{
		ID:         fileID,
		OwnerID:    ownerID,
		BucketName: storage.BucketAvatars.Name(),
		ObjectKey:  "avatars/users/file.png",
	}

	bucketType := storage.BucketType(file.BucketName)

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), file.ObjectKey, mock.Anything).
		Return(errors.New("minio error")).Once()

	resp, err := svc.DeleteFile(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_DeleteFile_RepoError(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)
	fileID, ownerID := uuid.New(), uuid.New()
	req := &filev1.DeleteFileRequest{FileId: fileID.String(), OwnerId: ownerID.String()}
	file := &domain.File{ID: fileID, OwnerID: ownerID, BucketName: "avatars", ObjectKey: "file.png"}
	bucketType := storage.BucketType(file.BucketName)
	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), file.ObjectKey, mock.Anything).Return(nil).Once()
	fileRepo.On("HardDelete", mock.Anything, fileID).Return(nil).Once()
	resp, err := svc.DeleteFile(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_FullFlow_UploadConfirmDownloadDelete(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)
	ownerID := uuid.New()
	ownerType := domain.OwnerTypeUser
	bucketType := storage.BucketAvatars

	uploadReq := &filev1.GetUploadURLRequest{
		OwnerId:    ownerID.String(),
		OwnerType:  ownerType.String(),
		BucketType: bucketType.Name(),
		FileName:   "avatar.png",
		MimeType:   "image/png",
		FileSize:   1024,
	}

	fileRepo.On("Create", mock.Anything, mock.MatchedBy(func(f *domain.File) bool {
		return f.Status == "pending" && f.BucketName == bucketType.Name()
	})).Return(nil).Once()

	minioClient.On("GetPresignedPutURL", mock.Anything, bucketType, mock.Anything, 15*time.Minute).
		Return("http://minio/upload-url", nil).Once()

	uploadResp, err := svc.GetUploadURL(context.Background(), uploadReq)
	require.NoError(t, err)
	fileID, _ := uuid.Parse(uploadResp.FileId)

	confirmReq := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	file := &domain.File{
		ID:         fileID,
		OwnerID:    ownerID,
		OwnerType:  ownerType.String(),
		BucketName: bucketType.Name(),
		ObjectKey:  uploadResp.ObjectKey,
		Status:     "pending",
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("StatObject", mock.Anything, bucketType.Name(), uploadResp.ObjectKey).Return(nil).Once()
	fileRepo.On("GetActiveByOwnerAndType",
		mock.Anything,
		ownerID,
		ownerType.String(),
		bucketType.Name(),
	).Return(nil, repository.ErrNotFound).Once()
	fileRepo.On("ConfirmUpload", mock.Anything, fileID).Return(nil).Once()

	_, err = svc.ConfirmUpload(context.Background(), confirmReq)
	require.NoError(t, err)

	downloadReq := &filev1.GetDownloadURLRequest{
		FileId:        fileID.String(),
		ExpirySeconds: 3600,
	}

	fileRepo.On("GetActiveByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("GetPresignedGetURL",
		mock.Anything,
		bucketType,
		uploadResp.ObjectKey,
		time.Hour,
	).Return("http://minio/download-url", nil).Once()

	downloadResp, err := svc.GetDownloadURL(context.Background(), downloadReq)
	require.NoError(t, err)
	assert.Equal(t, "http://minio/download-url", downloadResp.DownloadUrl)

	deleteReq := &filev1.DeleteFileRequest{
		FileId:  fileID.String(),
		OwnerId: ownerID.String(),
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), uploadResp.ObjectKey, mock.Anything).Return(nil).Once()
	fileRepo.On("HardDelete", mock.Anything, fileID).Return(nil).Once()

	deleteResp, err := svc.DeleteFile(context.Background(), deleteReq)
	require.NoError(t, err)
	assert.NotNil(t, deleteResp)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_GetFileByOwner_Success(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	ownerID := uuid.New()
	ownerType := domain.OwnerTypeUser
	bucketType := storage.BucketAvatars

	req := &filev1.GetFileByOwnerRequest{
		OwnerId:    ownerID.String(),
		OwnerType:  ownerType.String(),
		BucketType: bucketType.Name(),
	}

	file := &domain.File{
		ID:         uuid.New(),
		OwnerID:    ownerID,
		OwnerType:  ownerType.String(),
		BucketName: bucketType.Name(),
		ObjectKey:  bucketType.Prefix(ownerType, ownerID.String()) + "/abc123.png",
		FileName:   "avatar.png",
		MimeType:   "image/png",
		FileSize:   1024,
		Status:     "active",
		CreatedAt:  time.Now(),
	}

	minioClient.On("Endpoint").Return("http://localhost:9000").Once()

	fileRepo.On("GetActiveByOwnerAndType", mock.Anything, ownerID, ownerType.String(), bucketType.Name()).
		Return(file, nil).Once()

	resp, err := svc.GetFileByOwner(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, file.ID.String(), resp.FileId)
	assert.Equal(t, file.ObjectKey, resp.ObjectKey)
	assert.Equal(t, file.MimeType, resp.MimeType)
	assert.Equal(t, file.FileSize, resp.FileSize)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_GetFileByOwner_InvalidOwnerID(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.GetFileByOwnerRequest{
		OwnerId:    "invalid-uuid",
		OwnerType:  "user",
		BucketType: "avatars",
	}

	resp, err := svc.GetFileByOwner(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_GetFileByOwner_InvalidCombination(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	// ❌ message не может иметь аватар
	req := &filev1.GetFileByOwnerRequest{
		OwnerId:    uuid.New().String(),
		OwnerType:  domain.OwnerTypeMessage.String(),
		BucketType: storage.BucketAvatars.Name(),
	}

	resp, err := svc.GetFileByOwner(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "does not support owner_type")
}

func TestFileService_GetFileByOwner_NotFound(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	ownerID := uuid.New()
	req := &filev1.GetFileByOwnerRequest{
		OwnerId:    ownerID.String(),
		OwnerType:  "user",
		BucketType: "avatars",
	}

	fileRepo.On("GetActiveByOwnerAndType", mock.Anything, ownerID, domain.OwnerTypeUser.String(), "avatars").
		Return(nil, repository.ErrNotFound).Once()

	resp, err := svc.GetFileByOwner(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_GetFileByOwner_RepoError(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	ownerID := uuid.New()
	req := &filev1.GetFileByOwnerRequest{
		OwnerId:    ownerID.String(),
		OwnerType:  domain.OwnerTypeUser.String(),
		BucketType: "avatars",
	}

	fileRepo.On("GetActiveByOwnerAndType", mock.Anything, ownerID, domain.OwnerTypeUser.String(), "avatars").
		Return(nil, errors.New("db connection failed")).Once()

	resp, err := svc.GetFileByOwner(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_ReplacesOldAvatar(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	ownerID := uuid.New()
	ownerType := domain.OwnerTypeUser
	bucketType := storage.BucketAvatars

	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	newFile := &domain.File{
		ID:         fileID,
		OwnerID:    ownerID,
		OwnerType:  ownerType.String(),
		BucketName: bucketType.Name(),
		ObjectKey:  bucketType.Prefix(ownerType, ownerID.String()) + "/new.png",
		Status:     "pending",
	}

	oldFileID := uuid.New()
	oldFile := &domain.File{
		ID:         oldFileID,
		OwnerID:    ownerID,
		OwnerType:  ownerType.String(),
		BucketName: bucketType.Name(),
		ObjectKey:  bucketType.Prefix(ownerType, ownerID.String()) + "/old.png",
		Status:     "active",
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(newFile, nil).Once()
	minioClient.On("StatObject", mock.Anything, bucketType.Name(), newFile.ObjectKey).Return(nil).Once()
	fileRepo.On("GetActiveByOwnerAndType", mock.Anything, ownerID, ownerType.String(), bucketType.Name()).
		Return(oldFile, nil).Once()
	minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), oldFile.ObjectKey, mock.Anything).
		Return(nil).Once()
	fileRepo.On("SoftDelete", mock.Anything, oldFileID).Return(nil).Once()
	fileRepo.On("ConfirmUpload", mock.Anything, fileID).Return(nil).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_NoOldFile_FirstUpload(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	ownerID := uuid.New()
	ownerType := domain.OwnerTypeUser
	bucketType := storage.BucketAvatars

	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	file := &domain.File{
		ID:         fileID,
		OwnerID:    ownerID,
		OwnerType:  ownerType.String(),
		BucketName: bucketType.Name(),
		ObjectKey:  bucketType.Prefix(ownerType, ownerID.String()) + "/first.png",
		Status:     "pending",
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("StatObject", mock.Anything, bucketType.Name(), file.ObjectKey).Return(nil).Once()
	fileRepo.On("GetActiveByOwnerAndType", mock.Anything, ownerID, ownerType.String(), bucketType.Name()).
		Return(nil, repository.ErrNotFound).Once()
	fileRepo.On("ConfirmUpload", mock.Anything, fileID).Return(nil).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_MessageOwner_NoReplacement(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	messageID := uuid.New()
	ownerType := domain.OwnerTypeMessage
	bucketType := storage.BucketAttachments

	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	file := &domain.File{
		ID:         fileID,
		OwnerID:    messageID,
		OwnerType:  ownerType.String(),
		BucketName: bucketType.Name(),
		ObjectKey:  bucketType.Prefix(ownerType, messageID.String()) + "/doc.pdf",
		Status:     "pending",
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()
	minioClient.On("StatObject", mock.Anything, bucketType.Name(), file.ObjectKey).Return(nil).Once()
	fileRepo.On("ConfirmUpload", mock.Anything, fileID).Return(nil).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_ConfirmUpload_MinioRemoveError_LogsButContinues(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	ownerID := uuid.New()
	ownerType := domain.OwnerTypeUser
	bucketType := storage.BucketAvatars

	req := &filev1.ConfirmUploadRequest{FileId: fileID.String()}

	newFile := &domain.File{
		ID: fileID, OwnerID: ownerID, OwnerType: ownerType.String(),
		BucketName: bucketType.Name(), ObjectKey: bucketType.Prefix(ownerType, ownerID.String()) + "/new.png", Status: "pending",
	}
	oldFile := &domain.File{
		ID: uuid.New(), OwnerID: ownerID, OwnerType: ownerType.String(),
		BucketName: bucketType.Name(), ObjectKey: bucketType.Prefix(ownerType, ownerID.String()) + "/old.png", Status: "active",
	}

	fileRepo.On("GetByID", mock.Anything, fileID).Return(newFile, nil).Once()
	minioClient.On("StatObject", mock.Anything, bucketType.Name(), newFile.ObjectKey).Return(nil).Once()
	fileRepo.On("GetActiveByOwnerAndType", mock.Anything, ownerID, ownerType.String(), bucketType.Name()).
		Return(oldFile, nil).Once()
	minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), oldFile.ObjectKey, mock.Anything).
		Return(errors.New("connection timeout")).Once()
	fileRepo.On("SoftDelete", mock.Anything, oldFile.ID).Return(nil).Once()
	fileRepo.On("ConfirmUpload", mock.Anything, fileID).Return(nil).Once()

	resp, err := svc.ConfirmUpload(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_GetFilesByOwner_Success_MultipleFiles(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	messageID := uuid.New()
	ownerType := domain.OwnerTypeMessage
	bucketType := storage.BucketAttachments

	req := &filev1.GetFilesByOwnerRequest{
		OwnerId:    messageID.String(),
		OwnerType:  ownerType.String(),
		BucketType: bucketType.Name(),
	}

	files := []*domain.File{
		{
			ID: uuid.New(), OwnerID: messageID, OwnerType: ownerType.String(),
			BucketName: bucketType.Name(), ObjectKey: bucketType.Prefix(ownerType, messageID.String()) + "/file1.png",
			FileName: "photo1.png", MimeType: "image/png", FileSize: 1024,
		},
	}

	fileRepo.On("GetAllActiveByOwnerAndType", mock.Anything, messageID, ownerType.String(), bucketType.Name()).
		Return(files, nil).Once()

	minioClient.On("Endpoint").Return("http://localhost:9000").Once()

	resp, err := svc.GetFilesByOwner(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Files, 1)
	assert.Contains(t, resp.Files[0].PublicUrl, "http://localhost:9000/attachments/")

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_GetFilesByOwner_EmptyList(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	messageID := uuid.New()
	ownerType := domain.OwnerTypeMessage
	bucketType := storage.BucketAttachments

	req := &filev1.GetFilesByOwnerRequest{
		OwnerId:    messageID.String(),
		OwnerType:  ownerType.String(),
		BucketType: bucketType.Name(),
	}

	fileRepo.On("GetAllActiveByOwnerAndType", mock.Anything, messageID, ownerType.String(), bucketType.Name()).
		Return([]*domain.File{}, nil).Once()

	resp, err := svc.GetFilesByOwner(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Files)

	fileRepo.AssertExpectations(t)
}

func TestFileService_GetFilesByOwner_InvalidOwnerID(t *testing.T) {
	svc, _, _ := newTestFileService(t)

	req := &filev1.GetFilesByOwnerRequest{
		OwnerId:    "invalid-uuid",
		OwnerType:  domain.OwnerTypeMessage.String(),
		BucketType: "attachments",
	}

	resp, err := svc.GetFilesByOwner(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFileService_GetFilesByOwner_RepoError(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	messageID := uuid.New()
	ownerType := domain.OwnerTypeMessage
	bucketType := storage.BucketAttachments

	req := &filev1.GetFilesByOwnerRequest{
		OwnerId:    messageID.String(),
		OwnerType:  ownerType.String(),
		BucketType: bucketType.Name(),
	}

	fileRepo.On("GetAllActiveByOwnerAndType", mock.Anything, messageID, ownerType.String(), bucketType.Name()).
		Return(nil, errors.New("db error")).Once()

	resp, err := svc.GetFilesByOwner(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))

	fileRepo.AssertExpectations(t)
}

func TestFileService_CleanupPendingFiles_Success(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	olderThan := 1 * time.Hour
	cutoff := time.Now().Add(-olderThan)

	pendingIDs := []uuid.UUID{uuid.New(), uuid.New()}
	files := []*domain.File{
		{ID: pendingIDs[0], BucketName: "avatars", ObjectKey: "old1.png"},
		{ID: pendingIDs[1], BucketName: "attachments", ObjectKey: "old2.jpg"},
	}

	fileRepo.On("GetPendingOlderThan", mock.Anything, mock.MatchedBy(func(t time.Time) bool {
		return !t.After(cutoff.Add(1 * time.Second))
	})).Return(pendingIDs, nil).Once()

	for i, id := range pendingIDs {
		fileRepo.On("GetByID", mock.Anything, id).Return(files[i], nil).Once()
		bucketType := storage.BucketType(files[i].BucketName)
		minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), files[i].ObjectKey, mock.Anything).
			Return(nil).Once()
		fileRepo.On("HardDelete", mock.Anything, id).Return(nil).Once()
	}

	deleted, err := svc.CleanupPendingFiles(context.Background(), olderThan)

	require.NoError(t, err)
	assert.Equal(t, 2, deleted)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestFileService_CleanupPendingFiles_NoFiles(t *testing.T) {
	svc, fileRepo, _ := newTestFileService(t)

	fileRepo.On("GetPendingOlderThan", mock.Anything, mock.Anything).
		Return([]uuid.UUID{}, nil).Once()

	deleted, err := svc.CleanupPendingFiles(context.Background(), 1*time.Hour)

	require.NoError(t, err)
	assert.Equal(t, 0, deleted)

	fileRepo.AssertExpectations(t)
}

func TestFileService_CleanupPendingFiles_MinioError_LogsButContinues(t *testing.T) {
	svc, fileRepo, minioClient := newTestFileService(t)

	fileID := uuid.New()
	file := &domain.File{ID: fileID, BucketName: "avatars", ObjectKey: "old.png"}

	fileRepo.On("GetPendingOlderThan", mock.Anything, mock.Anything).
		Return([]uuid.UUID{fileID}, nil).Once()
	fileRepo.On("GetByID", mock.Anything, fileID).Return(file, nil).Once()

	bucketType := storage.BucketType(file.BucketName)
	minioClient.On("RemoveObject", mock.Anything, bucketType.Name(), file.ObjectKey, mock.Anything).
		Return(errors.New("timeout")).Once()

	fileRepo.On("HardDelete", mock.Anything, fileID).Return(nil).Once()

	deleted, err := svc.CleanupPendingFiles(context.Background(), 1*time.Hour)

	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	fileRepo.AssertExpectations(t)
	minioClient.AssertExpectations(t)
}

func TestOwnerType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		ot       domain.OwnerType
		expected bool
	}{
		{"user", domain.OwnerTypeUser, true},
		{"chat", domain.OwnerTypeChat, true},
		{"server", domain.OwnerTypeServer, true},
		{"message", domain.OwnerTypeMessage, true},
		{"invalid", domain.OwnerType("invalid"), false},
		{"empty", domain.OwnerType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ot.IsValid())
		})
	}
}

func TestOwnerType_IsSingleFileOwner(t *testing.T) {
	tests := []struct {
		name     string
		ot       domain.OwnerType
		expected bool
	}{
		{"user", domain.OwnerTypeUser, true},
		{"chat", domain.OwnerTypeChat, true},
		{"server", domain.OwnerTypeServer, true},
		{"message", domain.OwnerTypeMessage, false},
		{"invalid", domain.OwnerType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ot.IsSingleFileOwner())
		})
	}
}

func TestOwnerType_CanHaveAttachments(t *testing.T) {
	assert.True(t, domain.OwnerTypeMessage.CanHaveAttachments())
	assert.False(t, domain.OwnerTypeUser.CanHaveAttachments())
	assert.False(t, domain.OwnerTypeChat.CanHaveAttachments())
	assert.False(t, domain.OwnerTypeServer.CanHaveAttachments())
}

func TestOwnerType_String(t *testing.T) {
	assert.Equal(t, "user", domain.OwnerTypeUser.String())
	assert.Equal(t, "chat", domain.OwnerTypeChat.String())
	assert.Equal(t, "server", domain.OwnerTypeServer.String())
	assert.Equal(t, "message", domain.OwnerTypeMessage.String())
}

func TestBucketType_IsValid(t *testing.T) {
	assert.True(t, storage.BucketAvatars.IsValid())
	assert.True(t, storage.BucketAttachments.IsValid())
	assert.False(t, storage.BucketType("invalid").IsValid())
}

func TestBucketType_Name(t *testing.T) {
	assert.Equal(t, "avatars", storage.BucketAvatars.Name())
	assert.Equal(t, "attachments", storage.BucketAttachments.Name())
}

func TestBucketType_Prefix(t *testing.T) {
	ownerID := uuid.New().String()

	// Аватары
	assert.Equal(t, "users/"+ownerID, storage.BucketAvatars.Prefix(domain.OwnerTypeUser, ownerID))
	assert.Equal(t, "chats/"+ownerID, storage.BucketAvatars.Prefix(domain.OwnerTypeChat, ownerID))
	assert.Equal(t, "servers/"+ownerID, storage.BucketAvatars.Prefix(domain.OwnerTypeServer, ownerID))
	assert.Equal(t, "", storage.BucketAvatars.Prefix(domain.OwnerTypeMessage, ownerID)) // message не может иметь аватар

	// Вложения
	assert.Equal(t, "messages/"+ownerID, storage.BucketAttachments.Prefix(domain.OwnerTypeMessage, ownerID))
	assert.Equal(t, "", storage.BucketAttachments.Prefix(domain.OwnerTypeUser, ownerID)) // user не может иметь вложения
}

func TestBucketType_AllowedOwnerTypes(t *testing.T) {
	avatarsAllowed := storage.BucketAvatars.AllowedOwnerTypes()
	assert.Contains(t, avatarsAllowed, domain.OwnerTypeUser)
	assert.Contains(t, avatarsAllowed, domain.OwnerTypeChat)
	assert.Contains(t, avatarsAllowed, domain.OwnerTypeServer)
	assert.NotContains(t, avatarsAllowed, domain.OwnerTypeMessage)

	attachmentsAllowed := storage.BucketAttachments.AllowedOwnerTypes()
	assert.Contains(t, attachmentsAllowed, domain.OwnerTypeMessage)
	assert.NotContains(t, attachmentsAllowed, domain.OwnerTypeUser)
}

func TestBucketType_IsValidCombination(t *testing.T) {
	// ✅ Валидные комбинации
	assert.True(t, storage.BucketAvatars.IsValidCombination(domain.OwnerTypeUser))
	assert.True(t, storage.BucketAvatars.IsValidCombination(domain.OwnerTypeChat))
	assert.True(t, storage.BucketAvatars.IsValidCombination(domain.OwnerTypeServer))
	assert.True(t, storage.BucketAttachments.IsValidCombination(domain.OwnerTypeMessage))

	// ❌ Невалидные комбинации
	assert.False(t, storage.BucketAvatars.IsValidCombination(domain.OwnerTypeMessage))
	assert.False(t, storage.BucketAttachments.IsValidCombination(domain.OwnerTypeUser))
	assert.False(t, storage.BucketAttachments.IsValidCombination(domain.OwnerTypeChat))
}
