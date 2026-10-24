package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/google/uuid"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *domain.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error)
	GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error)
	GetByChatAfter(ctx context.Context, chatID uuid.UUID, after time.Time, limit int) ([]*domain.Message, error)
	Update(ctx context.Context, id uuid.UUID, newContent string) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type messageRepo struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) MessageRepository {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, msg *domain.Message) error {
	if !domain.IsValidContentType(msg.ContentType) {
		return fmt.Errorf("invalid content type: %s", msg.ContentType)
	}

	query := `
		INSERT INTO messages (
			chat_id, sender_id, content, content_type, reply_to_id
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		msg.ChatID,
		msg.SenderID,
		msg.Content,
		msg.ContentType,
		msg.ReplyToID,
	).Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

func (r *messageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, content, content_type, 
		       reply_to_id, created_at, updated_at, deleted_at
		FROM messages
		WHERE id = $1 AND deleted_at IS NULL
	`

	msg := &domain.Message{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID,
		&msg.ChatID,
		&msg.SenderID,
		&msg.Content,
		&msg.ContentType,
		&msg.ReplyToID,
		&msg.CreatedAt,
		&msg.UpdatedAt,
		&msg.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message by id: %w", err)
	}

	return msg, nil
}

func (r *messageRepo) GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, chat_id, sender_id, content, content_type, 
		       reply_to_id, created_at, updated_at, deleted_at
		FROM messages
		WHERE chat_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Content,
			&msg.ContentType,
			&msg.ReplyToID,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&msg.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

func (r *messageRepo) GetByChatAfter(ctx context.Context, chatID uuid.UUID, after time.Time, limit int) ([]*domain.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, chat_id, sender_id, content, content_type, 
		       reply_to_id, created_at, updated_at, deleted_at
		FROM messages
		WHERE chat_id = $1 
		  AND deleted_at IS NULL 
		  AND created_at > $2
		ORDER BY created_at ASC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, chatID, after, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages after time: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Content,
			&msg.ContentType,
			&msg.ReplyToID,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&msg.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

func (r *messageRepo) Update(ctx context.Context, id uuid.UUID, newContent string) error {
	query := `
		UPDATE messages
		SET content = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, newContent, id)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
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

func (r *messageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE messages
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
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

func (r *messageRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM messages 
			WHERE id = $1 AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check message exists: %w", err)
	}

	return exists, nil
}
