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

type InviteRepository interface {
	Create(ctx context.Context, invite *domain.ChatInvite) error
	GetByToken(ctx context.Context, token string) (*domain.ChatInvite, error)
	GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.ChatInvite, error)
	IncrementUses(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, token string) (bool, error)
}

type inviteRepo struct {
	db     *sql.DB
	schema string
}

func NewInviteRepository(db *sql.DB, schema string) InviteRepository {
	return &inviteRepo{
		db:     db,
		schema: schema,
	}
}

func (r *inviteRepo) Create(ctx context.Context, invite *domain.ChatInvite) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.chat_invites (id, chat_id, created_by, token, expires_at, max_uses, uses_count, created_at, used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, r.schema)

	_, err := r.db.ExecContext(ctx, query,
		invite.ID, invite.ChatID, invite.CreatedBy, invite.Token, invite.ExpiresAt,
		invite.MaxUses, invite.UsesCount, invite.CreatedAt, invite.UsedAt,
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
		return fmt.Errorf("failed to create invite: %w", err)
	}

	return nil
}

func (r *inviteRepo) GetByToken(ctx context.Context, token string) (*domain.ChatInvite, error) {
	query := fmt.Sprintf(`
		SELECT id, chat_id, created_by, token, expires_at, max_uses, uses_count, created_at, used_at
		FROM %s.chat_invites WHERE token = $1
	`, r.schema)

	invite := &domain.ChatInvite{}
	var expiresAt, usedAt sql.NullTime
	var maxUses sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&invite.ID, &invite.ChatID, &invite.CreatedBy, &invite.Token,
		&expiresAt, &maxUses, &invite.UsesCount, &invite.CreatedAt, &usedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invite by token: %w", err)
	}

	if expiresAt.Valid {
		invite.ExpiresAt = &expiresAt.Time
	}
	if maxUses.Valid {
		val := int(maxUses.Int64)
		invite.MaxUses = &val
	}
	if usedAt.Valid {
		invite.UsedAt = &usedAt.Time
	}

	return invite, nil
}

func (r *inviteRepo) GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.ChatInvite, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := fmt.Sprintf(`
		SELECT id, chat_id, created_by, token, expires_at, max_uses, uses_count, created_at, used_at
		FROM %s.chat_invites WHERE chat_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, r.schema)

	rows, err := r.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query invites by chat: %w", err)
	}
	defer rows.Close()

	return r.scanInvites(rows)
}

func (r *inviteRepo) IncrementUses(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`
		UPDATE %s.chat_invites SET uses_count = uses_count + 1,
		    used_at = CASE WHEN max_uses = 1 THEN $2 ELSE used_at END
		WHERE id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		  AND (max_uses IS NULL OR uses_count < max_uses)
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment invite uses: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrInviteNotFound
	}

	return nil
}

func (r *inviteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s.chat_invites WHERE id = $1`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete invite: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrInviteNotFound
	}

	return nil
}

func (r *inviteRepo) Exists(ctx context.Context, token string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s.chat_invites WHERE token = $1)`, r.schema)

	var exists bool
	err := r.db.QueryRowContext(ctx, query, token).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check invite exists: %w", err)
	}

	return exists, nil
}

func (r *inviteRepo) scanInvites(rows *sql.Rows) ([]*domain.ChatInvite, error) {
	invites := make([]*domain.ChatInvite, 0)
	for rows.Next() {
		invite := &domain.ChatInvite{}
		var expiresAt, usedAt sql.NullTime
		var maxUses sql.NullInt64

		err := rows.Scan(&invite.ID, &invite.ChatID, &invite.CreatedBy, &invite.Token,
			&expiresAt, &maxUses, &invite.UsesCount, &invite.CreatedAt, &usedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invite row: %w", err)
		}

		if expiresAt.Valid {
			invite.ExpiresAt = &expiresAt.Time
		}
		if maxUses.Valid {
			val := int(maxUses.Int64)
			invite.MaxUses = &val
		}
		if usedAt.Valid {
			invite.UsedAt = &usedAt.Time
		}

		invites = append(invites, invite)
	}

	return invites, nil
}
