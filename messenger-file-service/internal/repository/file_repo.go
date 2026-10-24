package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-file-service/internal/domain"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error)
	GetActiveByID(ctx context.Context, id uuid.UUID) (*domain.File, error)
	GetActiveByOwnerAndType(ctx context.Context, ownerID uuid.UUID, ownerType, bucketType string) (*domain.File, error)
	GetAllActiveByOwnerAndType(ctx context.Context, ownerID uuid.UUID, ownerType, bucketType string) ([]*domain.File, error)
	ConfirmUpload(ctx context.Context, fileID uuid.UUID) error
	SoftDelete(ctx context.Context, fileID uuid.UUID) error
	HardDelete(ctx context.Context, fileID uuid.UUID) error
	GetPendingOlderThan(ctx context.Context, olderThan time.Time) ([]uuid.UUID, error)
}

type fileRepo struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) FileRepository {
	return &fileRepo{db: db}
}

func (r *fileRepo) Create(ctx context.Context, file *domain.File) error {
	query := `
		INSERT INTO files (
			id, owner_id, owner_type, bucket_name, object_key, 
			file_name, mime_type, file_size, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
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
	if err != nil {
		return fmt.Errorf("failed to create file record: %w", err)
	}
	return nil
}

func (r *fileRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	query := `
		SELECT id, owner_id, owner_type, bucket_name, object_key, 
		       file_name, mime_type, file_size, status, created_at, 
		       confirmed_at, deleted_at
		FROM files
		WHERE id = $1
	`

	var f domain.File
	var confirmedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID,
		&f.OwnerID,
		&f.OwnerType,
		&f.BucketName,
		&f.ObjectKey,
		&f.FileName,
		&f.MimeType,
		&f.FileSize,
		&f.Status,
		&f.CreatedAt,
		&confirmedAt,
		&f.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file by id: %w", err)
	}

	if confirmedAt.Valid {
		f.ConfirmedAt = &confirmedAt.Time
	}

	return &f, nil
}

func (r *fileRepo) GetActiveByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	query := `
		SELECT id, owner_id, owner_type, bucket_name, object_key, 
		       file_name, mime_type, file_size, status, created_at, 
		       confirmed_at, deleted_at
		FROM files
		WHERE id = $1 
		  AND status = 'active' 
		  AND deleted_at IS NULL
	`

	var f domain.File
	var confirmedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID, &f.OwnerID, &f.OwnerType, &f.BucketName, &f.ObjectKey,
		&f.FileName, &f.MimeType, &f.FileSize, &f.Status, &f.CreatedAt,
		&confirmedAt, &f.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active file: %w", err)
	}

	if confirmedAt.Valid {
		f.ConfirmedAt = &confirmedAt.Time
	}

	return &f, nil
}

func (r *fileRepo) GetActiveByOwnerAndType(ctx context.Context, ownerID uuid.UUID, ownerType, bucketType string) (*domain.File, error) {
	query := `
		SELECT id, owner_id, owner_type, bucket_name, object_key, 
		       file_name, mime_type, file_size, status, created_at, 
		       confirmed_at, deleted_at
		FROM files 
		WHERE owner_id = $1 
		  AND owner_type = $2 
		  AND bucket_name = $3 
		  AND status = 'active'
		  AND deleted_at IS NULL
		ORDER BY created_at DESC 
		LIMIT 1
	`

	var f domain.File
	var confirmedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, ownerID, ownerType, bucketType).Scan(
		&f.ID, &f.OwnerID, &f.OwnerType, &f.BucketName, &f.ObjectKey,
		&f.FileName, &f.MimeType, &f.FileSize, &f.Status, &f.CreatedAt,
		&confirmedAt, &f.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active file by owner: %w", err)
	}

	if confirmedAt.Valid {
		f.ConfirmedAt = &confirmedAt.Time
	}

	return &f, nil
}

func (r *fileRepo) GetAllActiveByOwnerAndType(ctx context.Context, ownerID uuid.UUID, ownerType, bucketType string) ([]*domain.File, error) {
	query := `
		SELECT id, owner_id, owner_type, bucket_name, object_key, 
		       file_name, mime_type, file_size, status, created_at, 
		       confirmed_at, deleted_at
		FROM files 
		WHERE owner_id = $1 
		  AND owner_type = $2 
		  AND bucket_name = $3 
		  AND status = 'active'
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID, ownerType, bucketType)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []*domain.File
	for rows.Next() {
		var f domain.File
		var confirmedAt sql.NullTime

		err := rows.Scan(
			&f.ID, &f.OwnerID, &f.OwnerType, &f.BucketName, &f.ObjectKey,
			&f.FileName, &f.MimeType, &f.FileSize, &f.Status, &f.CreatedAt,
			&confirmedAt, &f.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file row: %w", err)
		}

		if confirmedAt.Valid {
			f.ConfirmedAt = &confirmedAt.Time
		}

		files = append(files, &f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating file rows: %w", err)
	}

	return files, nil
}

func (r *fileRepo) ConfirmUpload(ctx context.Context, fileID uuid.UUID) error {
	query := `
		UPDATE files
		SET status = 'active', confirmed_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`

	result, err := r.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("failed to confirm upload: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("file not found or already confirmed: %w", ErrNotFound)
	}

	return nil
}

func (r *fileRepo) SoftDelete(ctx context.Context, fileID uuid.UUID) error {
	query := `
		UPDATE files
		SET deleted_at = NOW(), status = 'deleted'
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("failed to soft delete file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *fileRepo) HardDelete(ctx context.Context, fileID uuid.UUID) error {
	query := `DELETE FROM files WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("failed to hard delete file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *fileRepo) GetPendingOlderThan(ctx context.Context, olderThan time.Time) ([]uuid.UUID, error) {
	query := `
		SELECT id FROM files 
		WHERE status = 'pending' 
		  AND created_at < $1 
		  AND deleted_at IS NULL
	`

	rows, err := r.db.QueryContext(ctx, query, olderThan)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending files: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan file id: %w", err)
		}
		ids = append(ids, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending file rows: %w", err)
	}

	return ids, nil
}
