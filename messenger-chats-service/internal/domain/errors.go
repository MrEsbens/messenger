package domain

import "errors"

// ───────────────────────────────────────────────────────────
// Общие ошибки доменного слоя
// ───────────────────────────────────────────────────────────
var (
	ErrNotFound         = errors.New("entity not found")
	ErrAlreadyExists    = errors.New("entity already exists")
	ErrInvalidInput     = errors.New("invalid input")
	ErrPermissionDenied = errors.New("permission denied")
	ErrInternal         = errors.New("internal error")
)

// ───────────────────────────────────────────────────────────
// Ошибки валидации
// ───────────────────────────────────────────────────────────
var (
	ErrValidationFailed = errors.New("validation failed")
	ErrRequiredField    = errors.New("required field is empty")
	ErrTooLong          = errors.New("field exceeds maximum length")
	ErrTooShort         = errors.New("field is below minimum length")
	ErrInvalidFormat    = errors.New("invalid format")
)

// ───────────────────────────────────────────────────────────
// Ошибки бизнес-логики
// ───────────────────────────────────────────────────────────
var (
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrOperationNotAllowed    = errors.New("operation not allowed")
	ErrResourceLocked         = errors.New("resource is locked")
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")
)
