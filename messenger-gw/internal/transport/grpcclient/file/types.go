package file

import (
	"time"

	"github.com/google/uuid"
)

type GetUploadURLRequest struct {
	OwnerID    uuid.UUID
	OwnerType  string
	BucketType string
	FileName   string
	MimeType   string
	FileSize   int64
}

type GetUploadURLResponse struct {
	FileID    uuid.UUID
	UploadURL string
	ObjectKey string
	ExpiresAt time.Time
}

type ConfirmUploadRequest struct {
	FileID uuid.UUID
}

type GetDownloadURLRequest struct {
	FileID        uuid.UUID
	ExpirySeconds int64
}

type GetDownloadURLResponse struct {
	DownloadURL string
	ExpiresAt   time.Time
}

type DeleteFileRequest struct {
	FileID  uuid.UUID
	OwnerID uuid.UUID
}

type GetFileByOwnerRequest struct {
	OwnerID    uuid.UUID
	OwnerType  string
	BucketType string
}

type GetFileByOwnerResponse struct {
	FileID    uuid.UUID
	ObjectKey string
	PublicURL string
	MimeType  string
	FileSize  int64
	FileName  string
}

type FileInfo struct {
	FileID    uuid.UUID
	ObjectKey string
	PublicURL string
	MimeType  string
	FileSize  int64
	FileName  string
}

type GetFilesByOwnerRequest struct {
	OwnerID    uuid.UUID
	OwnerType  string
	BucketType string
}

type GetFilesByOwnerResponse struct {
	Files []FileInfo
}
