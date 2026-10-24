package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"

	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Search(ctx context.Context, query string, limit, offset int) ([]*domain.User, error)
}

type userRepo struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO identity.users (id, username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			return errors.New("user with this email or username already exists")
		}
		return err
	}

	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, status, created_at, updated_at
		FROM identity.users
		WHERE id = $1
	`

	var u domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}

	return &u, err
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, status, created_at, updated_at
		FROM identity.users
		WHERE email = $1
	`

	var u domain.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}

	return &u, err
}

func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE identity.users
		SET username = COALESCE($1, username),
			status = COALESCE($2, status),
			updated_at = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query,
		user.Username,
		user.Status,
		user.UpdatedAt,
		user.ID,
	)
	return err
}

func (r *userRepo) Search(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	queryPattern := "%" + strings.ToLower(query) + "%"

	sql := `
		SELECT id, username, email, status, created_at, updated_at
		FROM identity.users
		WHERE 
			LOWER(username) LIKE $1 
			OR LOWER(email) LIKE $1
		ORDER BY username ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, sql, queryPattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.Status,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &u)
	}

	return users, rows.Err()
}
