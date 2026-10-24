package service_test

import (
	"context"
	"errors"
	"testing"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/MrEsbens/messenger-identity/internal/domain"
	repoMocks "github.com/MrEsbens/messenger-identity/internal/repository/mocks"
	"github.com/MrEsbens/messenger-identity/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestContactService(t *testing.T) (*service.ContactService, *repoMocks.MockUserRepository, *repoMocks.MockContactRepository) {
	userRepo := repoMocks.NewMockUserRepository(t)
	contactRepo := repoMocks.NewMockContactRepository(t)

	return service.NewContactService(userRepo, contactRepo), userRepo, contactRepo
}

func TestContactService_SendContact_Success(t *testing.T) {
	svc, userRepo, contactRepo := newTestContactService(t)

	userID := uuid.New()
	contactID := uuid.New()

	req := &identityv1.SendContactRequest{
		UserId:    userID.String(),
		ContactId: contactID.String(),
	}

	userRepo.On("GetByID", mock.Anything, contactID).Return(&domain.User{ID: contactID}, nil).Once()
	contactRepo.On("GetByPair", mock.Anything, userID, contactID).Return(nil, errors.New("not found")).Once()
	contactRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Contact) bool {
		return c.UserID == userID && c.ContactID == contactID && c.Status == domain.ContactStatusPending
	})).Return(nil).Once()

	resp, err := svc.SendContact(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "pending", resp.Status)
	assert.NotEmpty(t, resp.RequestId)

	userRepo.AssertExpectations(t)
	contactRepo.AssertExpectations(t)
}

func TestContactService_SendContact_Self(t *testing.T) {
	svc, _, _ := newTestContactService(t)
	id := uuid.New()

	req := &identityv1.SendContactRequest{UserId: id.String(), ContactId: id.String()}

	resp, err := svc.SendContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "cannot add yourself")
}

func TestContactService_SendContact_UserNotFound(t *testing.T) {
	svc, userRepo, _ := newTestContactService(t)

	userID := uuid.New()
	contactID := uuid.New()

	req := &identityv1.SendContactRequest{
		UserId:    userID.String(),
		ContactId: contactID.String(),
	}

	userRepo.On("GetByID", mock.Anything, contactID).Return(nil, errors.New("not found")).Once()

	resp, err := svc.SendContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	userRepo.AssertExpectations(t)
}

func TestContactService_SendContact_Duplicate(t *testing.T) {
	svc, userRepo, contactRepo := newTestContactService(t)

	userID := uuid.New()
	contactID := uuid.New()

	req := &identityv1.SendContactRequest{
		UserId:    userID.String(),
		ContactId: contactID.String(),
	}

	userRepo.On("GetByID", mock.Anything, contactID).Return(&domain.User{ID: contactID}, nil).Once()
	contactRepo.On("GetByPair", mock.Anything, userID, contactID).Return(&domain.Contact{}, nil).Once()

	resp, err := svc.SendContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))

	userRepo.AssertExpectations(t)
	contactRepo.AssertExpectations(t)
}

func TestContactService_SendContact_InvalidUserID(t *testing.T) {
	svc, _, _ := newTestContactService(t)

	req := &identityv1.SendContactRequest{
		UserId:    "invalid-uuid",
		ContactId: uuid.New().String(),
	}

	resp, err := svc.SendContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestContactService_SendContact_InvalidContactID(t *testing.T) {
	svc, _, _ := newTestContactService(t)

	req := &identityv1.SendContactRequest{
		UserId:    uuid.New().String(),
		ContactId: "invalid-uuid",
	}

	resp, err := svc.SendContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestContactService_AcceptContact_Success(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userB := uuid.New()
	userA := uuid.New()

	originalReq := &domain.Contact{
		ID:        uuid.New(),
		UserID:    userA,
		ContactID: userB,
		Status:    domain.ContactStatusPending,
	}

	req := &identityv1.AcceptContactRequest{
		UserId:        userB.String(),
		RequestFromId: userA.String(),
	}

	contactRepo.On("GetByPair", mock.Anything, userA, userB).Return(originalReq, nil).Once()
	contactRepo.On("UpdateStatus", mock.Anything, originalReq.ID, domain.ContactStatusAccepted).Return(nil).Once()
	contactRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Contact) bool {
		return c.UserID == userB && c.ContactID == userA && c.Status == domain.ContactStatusAccepted
	})).Return(nil).Once()

	resp, err := svc.AcceptContact(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	contactRepo.AssertExpectations(t)
}

func TestContactService_AcceptContact_RequestNotFound(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userA := uuid.New()
	userB := uuid.New()

	req := &identityv1.AcceptContactRequest{
		UserId:        userB.String(),
		RequestFromId: userA.String(),
	}

	contactRepo.On("GetByPair", mock.Anything, userA, userB).Return(nil, errors.New("not found")).Once()

	resp, err := svc.AcceptContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	contactRepo.AssertExpectations(t)
}

func TestContactService_AcceptContact_NotPending(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userA := uuid.New()
	userB := uuid.New()

	originalReq := &domain.Contact{
		ID:        uuid.New(),
		UserID:    userA,
		ContactID: userB,
		Status:    domain.ContactStatusAccepted,
	}

	req := &identityv1.AcceptContactRequest{
		UserId:        userB.String(),
		RequestFromId: userA.String(),
	}

	contactRepo.On("GetByPair", mock.Anything, userA, userB).Return(originalReq, nil).Once()

	resp, err := svc.AcceptContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	contactRepo.AssertExpectations(t)
}

func TestContactService_AcceptContact_InvalidUserID(t *testing.T) {
	svc, _, _ := newTestContactService(t)

	req := &identityv1.AcceptContactRequest{
		UserId:        "invalid",
		RequestFromId: uuid.New().String(),
	}

	resp, err := svc.AcceptContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestContactService_RejectContact_Success(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userB := uuid.New()
	userA := uuid.New()

	originalReq := &domain.Contact{
		ID:        uuid.New(),
		UserID:    userA,
		ContactID: userB,
		Status:    domain.ContactStatusPending,
	}

	req := &identityv1.RejectContactRequest{
		UserId:        userB.String(),
		RequestFromId: userA.String(),
	}

	contactRepo.On("GetByPair", mock.Anything, userA, userB).Return(originalReq, nil).Once()
	contactRepo.On("SoftDelete", mock.Anything, userB, userA).Return(nil).Once()

	resp, err := svc.RejectContact(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	contactRepo.AssertExpectations(t)
}

func TestContactService_RejectContact_NotFound(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userA := uuid.New()
	userB := uuid.New()

	req := &identityv1.RejectContactRequest{
		UserId:        userB.String(),
		RequestFromId: userA.String(),
	}

	contactRepo.On("GetByPair", mock.Anything, userA, userB).Return(nil, errors.New("not found")).Once()

	resp, err := svc.RejectContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	contactRepo.AssertExpectations(t)
}

func TestContactService_ListContacts_Success(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userID := uuid.New()
	contacts := []*domain.User{
		{ID: uuid.New(), Username: "alice", Email: "alice@test.com", Status: "online"},
		{ID: uuid.New(), Username: "bob", Email: "bob@test.com", Status: "away"},
	}

	req := &identityv1.ListContactsRequest{UserId: userID.String()}

	contactRepo.On("ListAccepted", mock.Anything, userID).Return(contacts, nil).Once()

	resp, err := svc.ListContacts(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Contacts, 2)
	assert.Equal(t, "alice", resp.Contacts[0].Username)
	assert.Equal(t, "bob", resp.Contacts[1].Username)

	contactRepo.AssertExpectations(t)
}

func TestContactService_ListContacts_Empty(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userID := uuid.New()
	req := &identityv1.ListContactsRequest{UserId: userID.String()}

	contactRepo.On("ListAccepted", mock.Anything, userID).Return([]*domain.User{}, nil).Once()

	resp, err := svc.ListContacts(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Contacts)

	contactRepo.AssertExpectations(t)
}

func TestContactService_ListContacts_InvalidUserID(t *testing.T) {
	svc, _, _ := newTestContactService(t)

	req := &identityv1.ListContactsRequest{UserId: "invalid-uuid"}

	resp, err := svc.ListContacts(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestContactService_ListContacts_RepoError(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userID := uuid.New()
	req := &identityv1.ListContactsRequest{UserId: userID.String()}

	contactRepo.On("ListAccepted", mock.Anything, userID).Return(nil, errors.New("db error")).Once()

	resp, err := svc.ListContacts(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))

	contactRepo.AssertExpectations(t)
}

func TestContactService_ListPendingRequests_Success(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userID := uuid.New()
	requesters := []*domain.User{
		{ID: uuid.New(), Username: "alice", Email: "alice@test.com", Status: "online"},
		{ID: uuid.New(), Username: "bob", Email: "bob@test.com", Status: "away"},
	}

	req := &identityv1.ListPendingRequestsRequest{UserId: userID.String()}

	contactRepo.On("ListIncomingPending", mock.Anything, userID).Return(requesters, nil).Once()

	resp, err := svc.ListPendingRequests(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Requests, 2)
	assert.Equal(t, "alice", resp.Requests[0].Username)
	assert.Equal(t, "bob", resp.Requests[1].Username)

	contactRepo.AssertExpectations(t)
}

func TestContactService_ListPendingRequests_Empty(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userID := uuid.New()
	req := &identityv1.ListPendingRequestsRequest{UserId: userID.String()}

	contactRepo.On("ListIncomingPending", mock.Anything, userID).Return([]*domain.User{}, nil).Once()

	resp, err := svc.ListPendingRequests(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Requests)

	contactRepo.AssertExpectations(t)
}

func TestContactService_ListPendingRequests_InvalidUserID(t *testing.T) {
	svc, _, _ := newTestContactService(t)

	req := &identityv1.ListPendingRequestsRequest{UserId: "invalid-uuid"}

	resp, err := svc.ListPendingRequests(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestContactService_DeleteContact_Success(t *testing.T) {
	svc, _, contactRepo := newTestContactService(t)

	userID := uuid.New()
	contactID := uuid.New()

	req := &identityv1.DeleteContactRequest{
		UserId:    userID.String(),
		ContactId: contactID.String(),
	}

	contactRepo.On("SoftDelete", mock.Anything, userID, contactID).Return(nil).Once()

	resp, err := svc.DeleteContact(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	contactRepo.AssertExpectations(t)
}

func TestContactService_DeleteContact_InvalidUserID(t *testing.T) {
	svc, _, _ := newTestContactService(t)

	req := &identityv1.DeleteContactRequest{
		UserId:    "invalid-uuid",
		ContactId: uuid.New().String(),
	}

	resp, err := svc.DeleteContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestContactService_DeleteContact_InvalidContactID(t *testing.T) {
	svc, _, _ := newTestContactService(t)

	req := &identityv1.DeleteContactRequest{
		UserId:    uuid.New().String(),
		ContactId: "invalid-uuid",
	}

	resp, err := svc.DeleteContact(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
