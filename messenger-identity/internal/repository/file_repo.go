package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/google/uuid"
)

// FileRepository определяет контракт для работы с файлами
// Потом станет gRPC-клиентом к Storage Service
type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error)
	GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) (*domain.File, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ConfirmUpload(ctx context.Context, fileID uuid.UUID) error
}

type fileRepo struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) FileRepository {
	return &fileRepo{db: db}
}

func (r *fileRepo) Create(ctx context.Context, file *domain.File) error {
	// ⚠️ Fully qualified name: storage.files (не зависит от search_path!)
	query := `
		INSERT INTO storage.files (id, owner_id, owner_type, bucket_name, object_key, file_name, mime_type, file_size, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		file.ID,
		file.OwnerID,
		file.OwnerType,
		file.BucketName,
		file.ObjectKey,
		file.FileName,
		file.MimeType,
		file.FileSize,
		"pending",
	)
	return err
}

func (r *fileRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	query := `
		SELECT id, owner_id, owner_type, bucket_name, object_key, file_name, mime_type, file_size, created_at, deleted_at
		FROM storage.files
		WHERE id = $1 AND deleted_at IS NULL
	`

	var f domain.File
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID,
		&f.OwnerID,
		&f.OwnerType,
		&f.BucketName,
		&f.ObjectKey,
		&f.FileName,
		&f.MimeType,
		&f.FileSize,
		&f.CreatedAt,
		&f.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("file not found")
	}

	return &f, err
}

func (r *fileRepo) GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) (*domain.File, error) {
	query := `
		SELECT id, owner_id, owner_type, bucket_name, object_key, file_name, mime_type, file_size, created_at, deleted_at
		FROM storage.files
		WHERE owner_type = $1 AND owner_id = $2 AND deleted_at IS NULL
	`

	var f domain.File
	err := r.db.QueryRowContext(ctx, query, ownerType, ownerID).Scan(
		&f.ID,
		&f.OwnerID,
		&f.OwnerType,
		&f.BucketName,
		&f.ObjectKey,
		&f.FileName,
		&f.MimeType,
		&f.FileSize,
		&f.CreatedAt,
		&f.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("file not found")
	}

	return &f, err
}

func (r *fileRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE storage.files
		SET deleted_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *fileRepo) ConfirmUpload(ctx context.Context, fileID uuid.UUID) error {
	query := `
		UPDATE storage.files
		SET status = 'active', confirmed_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, fileID)
	return err
}
