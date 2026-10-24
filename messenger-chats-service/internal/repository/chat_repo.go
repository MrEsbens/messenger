package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ChatRepository interface {
	Create(ctx context.Context, chat *domain.Chat) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error)
	GetByType(ctx context.Context, chatType domain.ChatType, limit, offset int) ([]*domain.Chat, error)
	GetByCreatedBy(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Chat, error)
	Update(ctx context.Context, chat *domain.Chat) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type chatRepo struct {
	db     *sql.DB
	schema string
}

func NewChatRepository(db *sql.DB, schema string) ChatRepository {
	return &chatRepo{
		db:     db,
		schema: schema,
	}
}

func (r *chatRepo) Create(ctx context.Context, chat *domain.Chat) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.chats (id, type, name, description, config_id, created_by, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, r.schema)

	_, err := r.db.ExecContext(ctx, query,
		chat.ID, chat.Type, chat.Name, chat.Description, chat.ConfigID,
		chat.CreatedBy, chat.CreatedAt, chat.UpdatedAt, chat.DeletedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return ErrAlreadyExists
			}
			if pqErr.Code == "23503" {
				return ErrForeignKeyViolation
			}
		}
		return fmt.Errorf("failed to create chat: %w", err)
	}

	return nil
}

func (r *chatRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	query := fmt.Sprintf(`
		SELECT id, type, name, description, config_id, created_by, created_at, updated_at, deleted_at
		FROM %s.chats WHERE id = $1 AND deleted_at IS NULL
	`, r.schema)

	chat := &domain.Chat{}
	var name, description, configID sql.NullString
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&chat.ID, &chat.Type, &name, &description, &configID,
		&chat.CreatedBy, &chat.CreatedAt, &chat.UpdatedAt, &deletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrChatNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chat by id: %w", err)
	}

	if name.Valid {
		chat.Name = &name.String
	}
	if description.Valid {
		chat.Description = &description.String
	}
	if configID.Valid {
		uid, _ := uuid.Parse(configID.String)
		chat.ConfigID = &uid
	}
	if deletedAt.Valid {
		chat.DeletedAt = &deletedAt.Time
	}

	return chat, nil
}

func (r *chatRepo) GetByType(ctx context.Context, chatType domain.ChatType, limit, offset int) ([]*domain.Chat, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := fmt.Sprintf(`
		SELECT id, type, name, description, config_id, created_by, created_at, updated_at, deleted_at
		FROM %s.chats WHERE type = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, r.schema)

	rows, err := r.db.QueryContext(ctx, query, chatType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats by type: %w", err)
	}
	defer rows.Close()

	return r.scanChats(rows)
}

func (r *chatRepo) GetByCreatedBy(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Chat, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := fmt.Sprintf(`
		SELECT id, type, name, description, config_id, created_by, created_at, updated_at, deleted_at
		FROM %s.chats WHERE created_by = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, r.schema)

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats by created_by: %w", err)
	}
	defer rows.Close()

	return r.scanChats(rows)
}

func (r *chatRepo) Update(ctx context.Context, chat *domain.Chat) error {
	query := fmt.Sprintf(`
		UPDATE %s.chats SET name = $2, description = $3, config_id = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query, chat.ID, chat.Name, chat.Description, chat.ConfigID, chat.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update chat: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrChatNotFound
	}

	return nil
}

func (r *chatRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`
		UPDATE %s.chats SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrChatNotFound
	}

	return nil
}

func (r *chatRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`
		SELECT EXISTS(SELECT 1 FROM %s.chats WHERE id = $1 AND deleted_at IS NULL)
	`, r.schema)

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check chat exists: %w", err)
	}

	return exists, nil
}

func (r *chatRepo) scanChats(rows *sql.Rows) ([]*domain.Chat, error) {
	chats := make([]*domain.Chat, 0)
	for rows.Next() {
		chat := &domain.Chat{}
		var name, description, configID sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(&chat.ID, &chat.Type, &name, &description, &configID,
			&chat.CreatedBy, &chat.CreatedAt, &chat.UpdatedAt, &deletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat row: %w", err)
		}

		if name.Valid {
			chat.Name = &name.String
		}
		if description.Valid {
			chat.Description = &description.String
		}
		if configID.Valid {
			uid, _ := uuid.Parse(configID.String)
			chat.ConfigID = &uid
		}
		if deletedAt.Valid {
			chat.DeletedAt = &deletedAt.Time
		}

		chats = append(chats, chat)
	}

	return chats, nil
}
