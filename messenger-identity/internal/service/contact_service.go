package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/MrEsbens/messenger-identity/internal/repository"
	"github.com/google/uuid"
)

type ContactService struct {
	identityv1.UnimplementedIdentityServiceServer

	userRepo    repository.UserRepository
	contactRepo repository.ContactRepository
}

func NewContactService(
	userRepo repository.UserRepository,
	contactRepo repository.ContactRepository,
) *ContactService {
	return &ContactService{
		userRepo:    userRepo,
		contactRepo: contactRepo,
	}
}

func (s *ContactService) SendContact(ctx context.Context, req *identityv1.SendContactRequest) (*identityv1.ContactRequestResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	if userID == contactID {
		return nil, status.Error(codes.InvalidArgument, "cannot add yourself")
	}

	_, err = s.userRepo.GetByID(ctx, contactID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "contact not found")
	}

	_, err = s.contactRepo.GetByPair(ctx, userID, contactID)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "contact request already exists")
	}

	contact := &domain.Contact{
		ID:        uuid.New(),
		UserID:    userID,
		ContactID: contactID,
		Status:    domain.ContactStatusPending,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, status.Error(codes.Internal, "failed to send request")
	}

	return &identityv1.ContactRequestResponse{
		RequestId: contact.ID.String(),
		Status:    string(contact.Status),
	}, nil
}

func (s *ContactService) AcceptContact(ctx context.Context, req *identityv1.AcceptContactRequest) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	requestFromID, err := uuid.Parse(req.RequestFromId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_from_id")
	}

	originalRequest, err := s.contactRepo.GetByPair(ctx, requestFromID, userID)
	if err != nil || originalRequest.Status != domain.ContactStatusPending {
		return nil, status.Error(codes.NotFound, "pending request not found")
	}

	if err := s.contactRepo.UpdateStatus(ctx, originalRequest.ID, domain.ContactStatusAccepted); err != nil {
		return nil, status.Error(codes.Internal, "failed to update request")
	}

	reverseContact := &domain.Contact{
		ID:        uuid.New(),
		UserID:    userID,
		ContactID: requestFromID,
		Status:    domain.ContactStatusAccepted,
	}
	if err := s.contactRepo.Create(ctx, reverseContact); err != nil {
		return nil, status.Error(codes.Internal, "failed to create reverse contact")
	}

	return &emptypb.Empty{}, nil
}

func (s *ContactService) RejectContact(ctx context.Context, req *identityv1.RejectContactRequest) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	requestFromID, err := uuid.Parse(req.RequestFromId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_from_id")
	}

	// Просто удаляем запись с запросом
	// В реальной системе можно делать Soft Delete или ставить статус 'rejected'
	// Здесь для простоты удаляем
	_, err = s.contactRepo.GetByPair(ctx, requestFromID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "request not found")
	}

	if err := s.contactRepo.SoftDelete(ctx, userID, requestFromID); err != nil {
		return nil, status.Error(codes.Internal, "failed to reject")
	}

	return &emptypb.Empty{}, nil
}

func (s *ContactService) ListPendingRequests(ctx context.Context, req *identityv1.ListPendingRequestsRequest) (*identityv1.ListPendingRequestsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	users, err := s.contactRepo.ListIncomingPending(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list requests")
	}

	dtoUsers := make([]*identityv1.UserDTO, 0, len(users))
	for _, u := range users {
		dtoUsers = append(dtoUsers, &identityv1.UserDTO{
			Id:        u.ID.String(),
			Username:  u.Username,
			Email:     u.Email,
			Status:    u.Status,
			CreatedAt: timestamppb.New(u.CreatedAt),
		})
	}

	return &identityv1.ListPendingRequestsResponse{Requests: dtoUsers}, nil
}

func (s *ContactService) ListContacts(ctx context.Context, req *identityv1.ListContactsRequest) (*identityv1.ListContactsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	users, err := s.contactRepo.ListAccepted(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list contacts")
	}

	dtoUsers := make([]*identityv1.UserDTO, 0, len(users))
	for _, u := range users {
		dtoUsers = append(dtoUsers, &identityv1.UserDTO{
			Id:        u.ID.String(),
			Username:  u.Username,
			Email:     u.Email,
			Status:    u.Status,
			CreatedAt: timestamppb.New(u.CreatedAt),
		})
	}

	return &identityv1.ListContactsResponse{Contacts: dtoUsers}, nil
}

func (s *ContactService) DeleteContact(ctx context.Context, req *identityv1.DeleteContactRequest) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	if err := s.contactRepo.SoftDelete(ctx, userID, contactID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete contact")
	}

	return &emptypb.Empty{}, nil
}
