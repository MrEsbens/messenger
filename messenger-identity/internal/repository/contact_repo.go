package repository

import (
	"context"
	"database/sql"

	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/google/uuid"
)

type ContactRepository interface {
	Create(ctx context.Context, contact *domain.Contact) error
	GetByPair(ctx context.Context, userID, contactID uuid.UUID) (*domain.Contact, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ContactStatus) error
	ListAccepted(ctx context.Context, userID uuid.UUID) ([]*domain.User, error)
	ListIncomingPending(ctx context.Context, userID uuid.UUID) ([]*domain.User, error)
	SoftDelete(ctx context.Context, userID, contactID uuid.UUID) error
}

type contactRepo struct {
	db       *sql.DB
	userRepo UserRepository
}

func NewContactRepository(db *sql.DB, userRepo UserRepository) ContactRepository {
	return &contactRepo{db: db, userRepo: userRepo}
}

func (r *contactRepo) Create(ctx context.Context, contact *domain.Contact) error {
	query := `
		INSERT INTO identity.contacts (id, user_id, contact_id, status, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.ExecContext(ctx, query, contact.ID, contact.UserID, contact.ContactID, contact.Status)
	return err
}

func (r *contactRepo) GetByPair(ctx context.Context, userID, contactID uuid.UUID) (*domain.Contact, error) {
	query := `SELECT id, user_id, contact_id, status, created_at, updated_at FROM identity.contacts WHERE user_id = $1 AND contact_id = $2`
	var c domain.Contact
	err := r.db.QueryRowContext(ctx, query, userID, contactID).Scan(&c.ID, &c.UserID, &c.ContactID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *contactRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ContactStatus) error {
	query := `UPDATE identity.contacts SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *contactRepo) ListAccepted(ctx context.Context, userID uuid.UUID) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.status, u.created_at
		FROM identity.contacts c
		JOIN identity.users u ON u.id = c.contact_id
		WHERE c.user_id = $1 AND c.status = 'accepted'
		ORDER BY c.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *contactRepo) ListIncomingPending(ctx context.Context, userID uuid.UUID) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.status, u.created_at
		FROM identity.contacts c
		JOIN identity.users u ON u.id = c.user_id
		WHERE c.contact_id = $1 AND c.status = 'pending'
		ORDER BY c.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *contactRepo) SoftDelete(ctx context.Context, userID, contactID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM identity.contacts WHERE (user_id=$1 AND contact_id=$2) OR (user_id=$2 AND contact_id=$1)`, userID, contactID)
	return err
}
