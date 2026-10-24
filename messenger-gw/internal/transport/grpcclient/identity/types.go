package identity

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Username string
	Email    string
	Password string
}

type LoginRequest struct {
	Email    string
	Password string
}

type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	User         UserDTO
}

type RefreshRequest struct {
	RefreshToken string
}

type RefreshResponse struct {
	AccessToken string
}

type UserDTO struct {
	ID        uuid.UUID
	Username  string
	Email     string
	AvatarURL string
	Status    string
	CreatedAt time.Time
}

type GetMeRequest struct {
	UserID uuid.UUID
}

type UserResponse struct {
	User UserDTO
}

type UpdateProfileRequest struct {
	UserID   uuid.UUID
	Username *string
	Status   *string
}

type SearchUsersRequest struct {
	Query  string
	Limit  int
	Offset int
}

type SearchUsersResponse struct {
	Users []UserDTO
}

type SendContactRequest struct {
	UserID    uuid.UUID
	ContactID uuid.UUID
}

type ContactRequestResponse struct {
	RequestID string
	Status    string
}

type AcceptContactRequest struct {
	UserID        uuid.UUID
	RequestFromID uuid.UUID
}

type RejectContactRequest struct {
	UserID        uuid.UUID
	RequestFromID uuid.UUID
}

type ListPendingRequestsRequest struct {
	UserID uuid.UUID
}

type ListPendingRequestsResponse struct {
	Requests []UserDTO
}

type DeleteContactRequest struct {
	UserID    uuid.UUID
	ContactID uuid.UUID
}

type ListContactsRequest struct {
	UserID uuid.UUID
}

type ListContactsResponse struct {
	Contacts []UserDTO
}

type GetUploadAvatarURLRequest struct {
	OwnerID  uuid.UUID
	FileName string
	MimeType string
	FileSize int64
}

type GetUploadURLResponse struct {
	FileID    uuid.UUID
	UploadURL string
	ObjectKey string
	ExpiresAt time.Time
}

type ConfirmUploadRequest struct {
	FileID uuid.UUID
}
