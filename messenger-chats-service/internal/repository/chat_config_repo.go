package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/google/uuid"
)

type ChatConfigRepository interface {
	Create(ctx context.Context, config *domain.ChatConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ChatConfig, error)
	Update(ctx context.Context, config *domain.ChatConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type chatConfigRepo struct {
	db     *sql.DB
	schema string
}

func NewChatConfigRepository(db *sql.DB, schema string) ChatConfigRepository {
	return &chatConfigRepo{
		db:     db,
		schema: schema,
	}
}

func (r *chatConfigRepo) Create(ctx context.Context, config *domain.ChatConfig) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.chat_configs (id, allow_member_invite, allow_member_kick, allow_member_mute,
			auto_moderation_enabled, max_message_length, default_notification_mode, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, r.schema)

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.AllowMemberInvite, config.AllowMemberKick, config.AllowMemberMute,
		config.AutoModerationEnabled, config.MaxMessageLength, config.DefaultNotificationMode,
		config.CreatedAt, config.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create chat config: %w", err)
	}

	return nil
}

func (r *chatConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ChatConfig, error) {
	query := fmt.Sprintf(`
		SELECT id, allow_member_invite, allow_member_kick, allow_member_mute,
		       auto_moderation_enabled, max_message_length, default_notification_mode, created_at, updated_at
		FROM %s.chat_configs WHERE id = $1
	`, r.schema)

	config := &domain.ChatConfig{}
	var notificationMode string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&config.ID, &config.AllowMemberInvite, &config.AllowMemberKick, &config.AllowMemberMute,
		&config.AutoModerationEnabled, &config.MaxMessageLength, &notificationMode,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chat config: %w", err)
	}

	config.DefaultNotificationMode = domain.NotificationMode(notificationMode)
	return config, nil
}

func (r *chatConfigRepo) Update(ctx context.Context, config *domain.ChatConfig) error {
	query := fmt.Sprintf(`
		UPDATE %s.chat_configs SET allow_member_invite = $2, allow_member_kick = $3, allow_member_mute = $4,
			auto_moderation_enabled = $5, max_message_length = $6, default_notification_mode = $7, updated_at = $8
		WHERE id = $1
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query,
		config.ID, config.AllowMemberInvite, config.AllowMemberKick, config.AllowMemberMute,
		config.AutoModerationEnabled, config.MaxMessageLength, config.DefaultNotificationMode, config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update chat config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *chatConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s.chat_configs WHERE id = $1`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete chat config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *chatConfigRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s.chat_configs WHERE id = $1)`, r.schema)

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check config exists: %w", err)
	}

	return exists, nil
}
