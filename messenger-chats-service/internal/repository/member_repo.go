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

type MemberRepository interface {
	Create(ctx context.Context, member *domain.ChatMember) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ChatMember, error)
	GetByChatAndUser(ctx context.Context, chatID, userID uuid.UUID) (*domain.ChatMember, error)
	GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.ChatMember, error)
	GetByUser(ctx context.Context, userID uuid.UUID, activeOnly bool, limit, offset int) ([]*domain.ChatMember, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role domain.MembershipRole) error
	UpdateNotificationMode(ctx context.Context, id uuid.UUID, mode domain.NotificationMode) error
	UpdateLastRead(ctx context.Context, id uuid.UUID, messageID uuid.UUID) error
	Leave(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, chatID, userID uuid.UUID) (bool, error)
	CountByChat(ctx context.Context, chatID uuid.UUID) (int, error)
}

type memberRepo struct {
	db     *sql.DB
	schema string
}

func NewMemberRepository(db *sql.DB, schema string) MemberRepository {
	return &memberRepo{
		db:     db,
		schema: schema,
	}
}

func (r *memberRepo) Create(ctx context.Context, member *domain.ChatMember) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.chat_members (id, chat_id, user_id, role, joined_at, left_at,
			notification_mode, is_muted_until, last_read_message_id, last_activity_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, r.schema)

	_, err := r.db.ExecContext(ctx, query,
		member.ID, member.ChatID, member.UserID, member.Role, member.JoinedAt, member.LeftAt,
		member.NotificationMode, member.IsMutedUntil, member.LastReadMessageID, member.LastActivityAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return ErrDuplicateMember
			}
			if pqErr.Code == "23503" {
				return ErrForeignKeyViolation
			}
		}
		return fmt.Errorf("failed to create member: %w", err)
	}

	return nil
}

func (r *memberRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ChatMember, error) {
	query := fmt.Sprintf(`
		SELECT id, chat_id, user_id, role, joined_at, left_at, notification_mode,
		       is_muted_until, last_read_message_id, last_activity_at
		FROM %s.chat_members WHERE id = $1
	`, r.schema)

	return r.scanMember(r.db.QueryRowContext(ctx, query, id))
}

func (r *memberRepo) GetByChatAndUser(ctx context.Context, chatID, userID uuid.UUID) (*domain.ChatMember, error) {
	query := fmt.Sprintf(`
		SELECT id, chat_id, user_id, role, joined_at, left_at, notification_mode,
		       is_muted_until, last_read_message_id, last_activity_at
		FROM %s.chat_members WHERE chat_id = $1 AND user_id = $2 AND left_at IS NULL
	`, r.schema)

	return r.scanMember(r.db.QueryRowContext(ctx, query, chatID, userID))
}

func (r *memberRepo) GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.ChatMember, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := fmt.Sprintf(`
		SELECT id, chat_id, user_id, role, joined_at, left_at, notification_mode,
		       is_muted_until, last_read_message_id, last_activity_at
		FROM %s.chat_members WHERE chat_id = $1 AND left_at IS NULL
		ORDER BY joined_at DESC LIMIT $2 OFFSET $3
	`, r.schema)

	rows, err := r.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query members by chat: %w", err)
	}
	defer rows.Close()

	return r.scanMembers(rows)
}

func (r *memberRepo) GetByUser(ctx context.Context, userID uuid.UUID, activeOnly bool, limit, offset int) ([]*domain.ChatMember, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var query string
	if activeOnly {
		query = fmt.Sprintf(`
			SELECT id, chat_id, user_id, role, joined_at, left_at, notification_mode,
			       is_muted_until, last_read_message_id, last_activity_at
			FROM %s.chat_members WHERE user_id = $1 AND left_at IS NULL
			ORDER BY joined_at DESC LIMIT $2 OFFSET $3
		`, r.schema)
	} else {
		query = fmt.Sprintf(`
			SELECT id, chat_id, user_id, role, joined_at, left_at, notification_mode,
			       is_muted_until, last_read_message_id, last_activity_at
			FROM %s.chat_members WHERE user_id = $1
			ORDER BY joined_at DESC LIMIT $2 OFFSET $3
		`, r.schema)
	}

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query members by user: %w", err)
	}
	defer rows.Close()

	return r.scanMembers(rows)
}

func (r *memberRepo) UpdateRole(ctx context.Context, id uuid.UUID, role domain.MembershipRole) error {
	query := fmt.Sprintf(`
		UPDATE %s.chat_members SET role = $2, is_muted_until = NULL
		WHERE id = $1 AND left_at IS NULL
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, role)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepo) UpdateNotificationMode(ctx context.Context, id uuid.UUID, mode domain.NotificationMode) error {
	query := fmt.Sprintf(`
		UPDATE %s.chat_members SET notification_mode = $2 WHERE id = $1 AND left_at IS NULL
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, mode)
	if err != nil {
		return fmt.Errorf("failed to update notification mode: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepo) UpdateLastRead(ctx context.Context, id uuid.UUID, messageID uuid.UUID) error {
	query := fmt.Sprintf(`
		UPDATE %s.chat_members SET last_read_message_id = $2, last_activity_at = $3
		WHERE id = $1 AND left_at IS NULL
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, messageID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update last read: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepo) Leave(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`
		UPDATE %s.chat_members SET left_at = $2 WHERE id = $1 AND left_at IS NULL
	`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to leave chat: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepo) Exists(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`
		SELECT EXISTS(SELECT 1 FROM %s.chat_members WHERE chat_id = $1 AND user_id = $2 AND left_at IS NULL)
	`, r.schema)

	var exists bool
	err := r.db.QueryRowContext(ctx, query, chatID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check member exists: %w", err)
	}

	return exists, nil
}

func (r *memberRepo) CountByChat(ctx context.Context, chatID uuid.UUID) (int, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.chat_members WHERE chat_id = $1 AND left_at IS NULL
	`, r.schema)

	var count int
	err := r.db.QueryRowContext(ctx, query, chatID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count members: %w", err)
	}

	return count, nil
}

func (r *memberRepo) scanMember(row *sql.Row) (*domain.ChatMember, error) {
	member := &domain.ChatMember{}
	var leftAt, isMutedUntil, lastActivityAt sql.NullTime
	var lastReadMessageID sql.NullString

	err := row.Scan(&member.ID, &member.ChatID, &member.UserID, &member.Role, &member.JoinedAt,
		&leftAt, &member.NotificationMode, &isMutedUntil, &lastReadMessageID, &lastActivityAt)

	if err == sql.ErrNoRows {
		return nil, ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan member row: %w", err)
	}

	if leftAt.Valid {
		member.LeftAt = &leftAt.Time
	}
	if isMutedUntil.Valid {
		member.IsMutedUntil = &isMutedUntil.Time
	}
	if lastReadMessageID.Valid {
		uid, _ := uuid.Parse(lastReadMessageID.String)
		member.LastReadMessageID = &uid
	}
	if lastActivityAt.Valid {
		member.LastActivityAt = &lastActivityAt.Time
	}

	return member, nil
}

func (r *memberRepo) scanMembers(rows *sql.Rows) ([]*domain.ChatMember, error) {
	members := make([]*domain.ChatMember, 0)
	for rows.Next() {
		member := &domain.ChatMember{}
		var leftAt, isMutedUntil, lastActivityAt sql.NullTime
		var lastReadMessageID sql.NullString

		err := rows.Scan(&member.ID, &member.ChatID, &member.UserID, &member.Role, &member.JoinedAt,
			&leftAt, &member.NotificationMode, &isMutedUntil, &lastReadMessageID, &lastActivityAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member row: %w", err)
		}

		if leftAt.Valid {
			member.LeftAt = &leftAt.Time
		}
		if isMutedUntil.Valid {
			member.IsMutedUntil = &isMutedUntil.Time
		}
		if lastReadMessageID.Valid {
			uid, _ := uuid.Parse(lastReadMessageID.String)
			member.LastReadMessageID = &uid
		}
		if lastActivityAt.Valid {
			member.LastActivityAt = &lastActivityAt.Time
		}

		members = append(members, member)
	}

	return members, nil
}
