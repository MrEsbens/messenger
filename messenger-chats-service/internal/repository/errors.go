package repository

import "errors"

var (
	// Общие ошибки репозитория
	ErrNotFound          = errors.New("entity not found")
	ErrAlreadyExists     = errors.New("entity already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrDatabaseOperation = errors.New("database operation failed")

	// Специфичные ошибки
	ErrChatNotFound        = errors.New("chat not found")
	ErrMemberNotFound      = errors.New("member not found")
	ErrInviteNotFound      = errors.New("invite not found")
	ErrDuplicateMember     = errors.New("user is already a member of this chat")
	ErrForeignKeyViolation = errors.New("foreign key violation")
)
