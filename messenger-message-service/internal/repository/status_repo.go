package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/google/uuid"
)

type MessageStatus struct {
	ID        uuid.UUID
	MessageID uuid.UUID
	UserID    uuid.UUID
	Status    string
	UpdatedAt time.Time
}

type StatusRepository interface {
	Upsert(ctx context.Context, status *MessageStatus) error
	GetByMessage(ctx context.Context, messageID uuid.UUID) ([]*MessageStatus, error)
	GetByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) (*MessageStatus, error)
	BulkUpdateByMessage(ctx context.Context, messageID uuid.UUID, status string, userIDs []uuid.UUID) error
}

type statusRepo struct {
	db *sql.DB
}

func NewStatusRepository(db *sql.DB) StatusRepository {
	return &statusRepo{db: db}
}

func (r *statusRepo) Upsert(ctx context.Context, status *MessageStatus) error {
	if !domain.IsValidMessageStatus(status.Status) {
		return fmt.Errorf("invalid message status: %s", status.Status)
	}

	query := `
		INSERT INTO message_statuses (message_id, user_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id) 
		DO UPDATE SET status = $3, updated_at = NOW()
		RETURNING id, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		status.MessageID,
		status.UserID,
		status.Status,
	).Scan(&status.ID, &status.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert message status: %w", err)
	}

	return nil
}

func (r *statusRepo) GetByMessage(ctx context.Context, messageID uuid.UUID) ([]*MessageStatus, error) {
	query := `
		SELECT id, message_id, user_id, status, updated_at
		FROM message_statuses
		WHERE message_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query message statuses: %w", err)
	}
	defer rows.Close()

	var statuses []*MessageStatus
	for rows.Next() {
		s := &MessageStatus{}
		err := rows.Scan(&s.ID, &s.MessageID, &s.UserID, &s.Status, &s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		statuses = append(statuses, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status rows: %w", err)
	}

	return statuses, nil
}

func (r *statusRepo) GetByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) (*MessageStatus, error) {
	query := `
		SELECT id, message_id, user_id, status, updated_at
		FROM message_statuses
		WHERE message_id = $1 AND user_id = $2
	`

	s := &MessageStatus{}
	err := r.db.QueryRowContext(ctx, query, messageID, userID).Scan(
		&s.ID, &s.MessageID, &s.UserID, &s.Status, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	return s, nil
}

func (r *statusRepo) BulkUpdateByMessage(ctx context.Context, messageID uuid.UUID, status string, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	if !domain.IsValidMessageStatus(status) {
		return fmt.Errorf("invalid message status: %s", status)
	}

	query := `
		INSERT INTO message_statuses (message_id, user_id, status)
		SELECT $1, unnest($3::uuid[]), $2
		ON CONFLICT (message_id, user_id) 
		DO UPDATE SET status = $2, updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, messageID, status, userIDs)
	if err != nil {
		return fmt.Errorf("failed to bulk update statuses: %w", err)
	}

	return nil
}
