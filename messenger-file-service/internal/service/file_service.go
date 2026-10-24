package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"github.com/MrEsbens/messenger-file-service/internal/domain"
	"github.com/MrEsbens/messenger-file-service/internal/pkg/minio"
	"github.com/MrEsbens/messenger-file-service/internal/pkg/storage"
	"github.com/MrEsbens/messenger-file-service/internal/repository"
	"github.com/google/uuid"
	MinioSDK "github.com/minio/minio-go/v7"
)

type FileService struct {
	filev1.UnimplementedFileServiceServer

	minioClient minio.MinIOClient
	fileRepo    repository.FileRepository

	maxFileSize int64
}

func NewFileService(minioClient minio.MinIOClient, fileRepo repository.FileRepository) *FileService {
	return &FileService{
		minioClient: minioClient,
		fileRepo:    fileRepo,
		maxFileSize: 5 * 1024 * 1024, // 5MB по умолчанию
	}
}

func (s *FileService) SetMaxFileSize(sizeBytes int64) {
	s.maxFileSize = sizeBytes
}

func (s *FileService) GetUploadURL(ctx context.Context, req *filev1.GetUploadURLRequest) (*filev1.GetUploadURLResponse, error) {
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
	}

	ownerType := domain.OwnerType(req.OwnerType)
	if !ownerType.IsValid() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner_type: %s", req.OwnerType)
	}

	bucketType := storage.BucketType(req.BucketType)
	if !bucketType.IsValid() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid bucket_type: %s", req.BucketType)
	}

	// Проверка: допустима ли комбинация бакета и owner_type
	if !bucketType.IsValidCombination(ownerType) {
		return nil, status.Errorf(codes.InvalidArgument, "bucket '%s' does not support owner_type '%s'",
			bucketType, ownerType)
	}

	if req.FileSize <= 0 || req.FileSize > s.maxFileSize {
		return nil, status.Errorf(codes.InvalidArgument, "file_size must be between 1 and %d bytes", s.maxFileSize)
	}

	// Генерация object_key с новым форматом префикса
	objectKey := s.generateObjectKey(bucketType, ownerType, ownerID, req.FileName)

	file := &domain.File{
		ID:         uuid.New(),
		OwnerID:    ownerID,
		OwnerType:  ownerType.String(),
		BucketName: bucketType.Name(),
		ObjectKey:  objectKey,
		FileName:   req.FileName,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		return nil, status.Error(codes.Internal, "failed to create file record")
	}

	uploadURL, err := s.minioClient.GetPresignedPutURL(ctx, bucketType, file.ObjectKey, 15*time.Minute)
	if err != nil {
		_ = s.fileRepo.SoftDelete(ctx, file.ID)
		return nil, status.Error(codes.Internal, "failed to generate upload URL")
	}

	return &filev1.GetUploadURLResponse{
		FileId:    file.ID.String(),
		UploadUrl: uploadURL,
		ObjectKey: file.ObjectKey,
		ExpiresAt: timestamppb.New(time.Now().Add(15 * time.Minute)),
	}, nil
}

func (s *FileService) ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*emptypb.Empty, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get file: %v", err)
	}

	if file.Status != "pending" {
		return nil, status.Errorf(codes.FailedPrecondition, "invalid file status: %s (expected: pending)", file.Status)
	}

	bucketName := storage.BucketType(file.BucketName).Name()
	if err := s.minioClient.StatObject(ctx, bucketName, file.ObjectKey); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "file not found in storage: %v", err)
	}

	ownerType := domain.OwnerType(file.OwnerType)
	if ownerType.IsSingleFileOwner() {
		if err := s.replaceOldFile(ctx, file); err != nil {
			log.Printf("⚠️ Failed to replace old file for owner %s: %v", file.OwnerID, err)
		}
	}

	if err := s.fileRepo.ConfirmUpload(ctx, fileID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to confirm upload: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *FileService) replaceOldFile(ctx context.Context, newFile *domain.File) error {
	oldFile, err := s.fileRepo.GetActiveByOwnerAndType(
		ctx,
		newFile.OwnerID,
		newFile.OwnerType,
		newFile.BucketName,
	)

	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to find old file: %w", err)
	}

	bucketName := storage.BucketType(oldFile.BucketName).Name()
	removeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = s.minioClient.RemoveObject(removeCtx, bucketName, oldFile.ObjectKey, MinioSDK.RemoveObjectOptions{})
	cancel()

	if err != nil {
		log.Printf("⚠️ Failed to remove old file %s from MinIO: %v", oldFile.ID, err)
	}

	if err := s.fileRepo.SoftDelete(ctx, oldFile.ID); err != nil {
		return fmt.Errorf("failed to soft delete old file: %w", err)
	}

	log.Printf("🗑️ Replaced file: old=%s, new=%s for owner %s", oldFile.ID, newFile.ID, newFile.OwnerID)
	return nil
}

func (s *FileService) GetDownloadURL(ctx context.Context, req *filev1.GetDownloadURLRequest) (*filev1.GetDownloadURLResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	file, err := s.fileRepo.GetActiveByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "file not found or not active")
		}
		return nil, status.Errorf(codes.Internal, "failed to get file: %v", err)
	}

	expiry := time.Duration(req.ExpirySeconds) * time.Second
	if expiry <= 0 {
		expiry = 1 * time.Hour // Дефолт: 1 час
	}
	if expiry > 24*time.Hour {
		expiry = 24 * time.Hour
	}

	bucketType := storage.BucketType(file.BucketName)
	downloadURL, err := s.minioClient.GetPresignedGetURL(ctx, bucketType, file.ObjectKey, expiry)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate download URL: %v", err)
	}

	return &filev1.GetDownloadURLResponse{
		DownloadUrl: downloadURL,
		ExpiresAt:   timestamppb.New(time.Now().Add(expiry)),
	}, nil
}

func (s *FileService) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*emptypb.Empty, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
	}

	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get file: %v", err)
	}

	if file.OwnerID != ownerID {
		return nil, status.Error(codes.PermissionDenied, "not allowed to delete this file")
	}

	bucketName := storage.BucketType(file.BucketName).Name()
	if err := s.minioClient.RemoveObject(ctx, bucketName, file.ObjectKey, MinioSDK.RemoveObjectOptions{}); err != nil {
		if minioErr, ok := err.(MinioSDK.ErrorResponse); ok && minioErr.Code == "NoSuchKey" {
			// Файл уже удалён — это ок
		} else {
			return nil, status.Errorf(codes.Internal, "failed to delete from storage: %v", err)
		}
	}

	if err := s.fileRepo.HardDelete(ctx, fileID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete from database: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *FileService) GetFileByOwner(ctx context.Context, req *filev1.GetFileByOwnerRequest) (*filev1.GetFileByOwnerResponse, error) {
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
	}

	ownerType := domain.OwnerType(req.OwnerType)
	bucketType := storage.BucketType(req.BucketType)

	if !bucketType.IsValidCombination(ownerType) {
		return nil, status.Errorf(codes.InvalidArgument, "bucket '%s' does not support owner_type '%s'",
			bucketType, ownerType)
	}

	file, err := s.fileRepo.GetActiveByOwnerAndType(ctx, ownerID, req.OwnerType, req.BucketType)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "file not found for this owner")
		}
		return nil, status.Errorf(codes.Internal, "failed to query file: %v", err)
	}

	publicURL := fmt.Sprintf("http://%s/%s/%s",
		s.minioClient.Endpoint(),
		file.BucketName,
		file.ObjectKey)

	return &filev1.GetFileByOwnerResponse{
		FileId:    file.ID.String(),
		ObjectKey: file.ObjectKey,
		PublicUrl: publicURL,
		MimeType:  file.MimeType,
		FileSize:  file.FileSize,
		FileName:  file.FileName,
	}, nil
}

func (s *FileService) GetFilesByOwner(ctx context.Context, req *filev1.GetFilesByOwnerRequest) (*filev1.GetFilesByOwnerResponse, error) {
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
	}

	ownerType := domain.OwnerType(req.OwnerType)
	bucketType := storage.BucketType(req.BucketType)

	// Проверка комбинации
	if !bucketType.IsValidCombination(ownerType) {
		return nil, status.Errorf(codes.InvalidArgument, "bucket '%s' does not support owner_type '%s'",
			bucketType, ownerType)
	}

	files, err := s.fileRepo.GetAllActiveByOwnerAndType(ctx, ownerID, req.OwnerType, req.BucketType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query files: %v", err)
	}

	result := make([]*filev1.FileInfo, 0, len(files))
	for _, f := range files {
		publicURL := fmt.Sprintf("http://%s/%s/%s",
			s.minioClient.Endpoint(),
			f.BucketName,
			f.ObjectKey)

		result = append(result, &filev1.FileInfo{
			FileId:    f.ID.String(),
			ObjectKey: f.ObjectKey,
			PublicUrl: publicURL,
			MimeType:  f.MimeType,
			FileSize:  f.FileSize,
			FileName:  f.FileName,
		})
	}

	return &filev1.GetFilesByOwnerResponse{
		Files: result,
	}, nil
}

// generateObjectKey строит ключ объекта в бакете с учётом owner_type
// Формат: {bucket_prefix}/{owner_type_plural}/{owner_id}/{uuid}{ext}
func (s *FileService) generateObjectKey(bucketType storage.BucketType, ownerType domain.OwnerType, ownerID uuid.UUID, fileName string) string {
	prefix := bucketType.Prefix(ownerType, ownerID.String())
	if prefix == "" {
		// Если префикс не удалось построить — это ошибка конфигурации
		// Возвращаем безопасный дефолт, чтобы не паниковать
		prefix = "unknown/" + ownerID.String()
	}

	ext := ".bin"
	if fileName != "" {
		for i := len(fileName) - 1; i >= 0; i-- {
			if fileName[i] == '.' {
				ext = fileName[i:]
				break
			}
		}
	}

	return prefix + "/" + uuid.New().String() + ext
}

func (s *FileService) CleanupPendingFiles(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)

	ids, err := s.fileRepo.GetPendingOlderThan(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to find pending files: %w", err)
	}

	deleted := 0
	for _, id := range ids {
		file, err := s.fileRepo.GetByID(ctx, id)
		if err != nil {
			log.Printf("⚠️ Failed to get file %s for cleanup: %v", id, err)
			continue
		}

		// 🔧 Конвертируем BucketType → string для MinIO
		bucketName := storage.BucketType(file.BucketName).Name()
		_ = s.minioClient.RemoveObject(ctx, bucketName, file.ObjectKey, MinioSDK.RemoveObjectOptions{})

		if err := s.fileRepo.HardDelete(ctx, id); err != nil {
			log.Printf("⚠️ Failed to delete file record %s: %v", id, err)
			continue
		}

		deleted++
	}

	if deleted > 0 {
		log.Printf("🧹 Cleaned up %d pending files older than %v", deleted, olderThan)
	}

	return deleted, nil
}