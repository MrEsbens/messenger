package service

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
)

// IdentityService — единая точка входа (Facade).
// Делегирует вызовы внутренним сервисам.
type IdentityService struct {
	identityv1.UnimplementedIdentityServiceServer

	auth    *AuthService
	user    *UserService
	contact *ContactService
}

func NewIdentityService(
	auth *AuthService,
	user *UserService,
	contact *ContactService,
) *IdentityService {
	return &IdentityService{
		auth:    auth,
		user:    user,
		contact: contact,
	}
}

// ───────────────────────────────────────────────────────────
// Auth Methods
// ───────────────────────────────────────────────────────────

func (s *IdentityService) Register(ctx context.Context, req *identityv1.RegisterRequest) (*identityv1.AuthResponse, error) {
	return s.auth.Register(ctx, req)
}

func (s *IdentityService) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.AuthResponse, error) {
	return s.auth.Login(ctx, req)
}

func (s *IdentityService) Refresh(ctx context.Context, req *identityv1.RefreshRequest) (*identityv1.RefreshResponse, error) {
	return s.auth.Refresh(ctx, req)
}

// ───────────────────────────────────────────────────────────
// User Methods
// ───────────────────────────────────────────────────────────

func (s *IdentityService) GetMe(ctx context.Context, req *identityv1.GetMeRequest) (*identityv1.UserResponse, error) {
	return s.user.GetMe(ctx, req)
}

func (s *IdentityService) UpdateProfile(ctx context.Context, req *identityv1.UpdateProfileRequest) (*identityv1.UserResponse, error) {
	return s.user.UpdateProfile(ctx, req)
}

func (s *IdentityService) GetUploadAvatarURL(ctx context.Context, req *identityv1.GetUploadAvatarURLRequest) (*identityv1.GetUploadURLResponse, error) {
	return s.user.GetUploadAvatarURL(ctx, req)
}

func (s *IdentityService) ConfirmAvatarUpload(ctx context.Context, req *identityv1.ConfirmUploadRequest) (*emptypb.Empty, error) {
	return s.user.ConfirmAvatarUpload(ctx, req)
}

func (s *IdentityService) SearchUsers(ctx context.Context, req *identityv1.SearchUsersRequest) (*identityv1.SearchUsersResponse, error) {
	return s.user.SearchUsers(ctx, req)
}

// ───────────────────────────────────────────────────────────
// Contact Methods
// ───────────────────────────────────────────────────────────

// SendContact — отправка запроса в друзья (ранее AddContact)
func (s *IdentityService) SendContact(ctx context.Context, req *identityv1.SendContactRequest) (*identityv1.ContactRequestResponse, error) {
	return s.contact.SendContact(ctx, req)
}

// AcceptContact — подтверждение входящего запроса
func (s *IdentityService) AcceptContact(ctx context.Context, req *identityv1.AcceptContactRequest) (*emptypb.Empty, error) {
	return s.contact.AcceptContact(ctx, req)
}

// RejectContact — отклонение входящего запроса
func (s *IdentityService) RejectContact(ctx context.Context, req *identityv1.RejectContactRequest) (*emptypb.Empty, error) {
	return s.contact.RejectContact(ctx, req)
}

func (s *IdentityService) ListPendingRequests(ctx context.Context, req *identityv1.ListPendingRequestsRequest) (*identityv1.ListPendingRequestsResponse, error) {
	return s.contact.ListPendingRequests(ctx, req)
}

// DeleteContact — удаление контакта или отмена запроса
func (s *IdentityService) DeleteContact(ctx context.Context, req *identityv1.DeleteContactRequest) (*emptypb.Empty, error) {
	return s.contact.DeleteContact(ctx, req)
}

// ListContacts — список подтверждённых друзей
func (s *IdentityService) ListContacts(ctx context.Context, req *identityv1.ListContactsRequest) (*identityv1.ListContactsResponse, error) {
	return s.contact.ListContacts(ctx, req)
}
