package identity

import (
	"context"
	"fmt"
	"strings"
	"time"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type IdentityClientInterface interface {
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	Refresh(ctx context.Context, req *RefreshRequest) (*RefreshResponse, error)
	GetMe(ctx context.Context, req *GetMeRequest) (*UserResponse, error)
	UpdateProfile(ctx context.Context, req *UpdateProfileRequest) (*UserResponse, error)
	SearchUsers(ctx context.Context, req *SearchUsersRequest) (*SearchUsersResponse, error)
	SendContact(ctx context.Context, req *SendContactRequest) (*ContactRequestResponse, error)
	AcceptContact(ctx context.Context, req *AcceptContactRequest) error
	RejectContact(ctx context.Context, req *RejectContactRequest) error
	ListPendingRequests(ctx context.Context, req *ListPendingRequestsRequest) (*ListPendingRequestsResponse, error)
	DeleteContact(ctx context.Context, req *DeleteContactRequest) error
	ListContacts(ctx context.Context, req *ListContactsRequest) (*ListContactsResponse, error)
	GetUploadAvatarURL(ctx context.Context, req *GetUploadAvatarURLRequest) (*GetUploadURLResponse, error)
	ConfirmAvatarUpload(ctx context.Context, req *ConfirmUploadRequest) error
	Close() error
}

type identityClient struct {
	client  identityv1.IdentityServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewIdentityClient(endpoint string, timeout time.Duration) (IdentityClientInterface, error) {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "grpc://")

	if endpoint == "" {
		return nil, fmt.Errorf("empty identity service endpoint")
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithReturnConnectionError(),
	}

	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to identity service: %w", err)
	}

	return &identityClient{
		client:  identityv1.NewIdentityServiceClient(conn),
		conn:    conn,
		timeout: timeout,
	}, nil
}

func (c *identityClient) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.Register(callCtx, &identityv1.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC Register failed: %w", err)
	}

	return &AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         protoToUserDTO(resp.User),
	}, nil
}

func (c *identityClient) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.Login(callCtx, &identityv1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC Login failed: %w", err)
	}

	return &AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         protoToUserDTO(resp.User),
	}, nil
}

func (c *identityClient) Refresh(ctx context.Context, req *RefreshRequest) (*RefreshResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.Refresh(callCtx, &identityv1.RefreshRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC Refresh failed: %w", err)
	}

	return &RefreshResponse{AccessToken: resp.AccessToken}, nil
}

func (c *identityClient) GetMe(ctx context.Context, req *GetMeRequest) (*UserResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetMe(callCtx, &identityv1.GetMeRequest{
		UserId: req.UserID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetMe failed: %w", err)
	}

	return &UserResponse{User: protoToUserDTO(resp.User)}, nil
}

func (c *identityClient) UpdateProfile(ctx context.Context, req *UpdateProfileRequest) (*UserResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &identityv1.UpdateProfileRequest{
		UserId: req.UserID.String(),
	}
	if req.Username != nil {
		grpcReq.Username = req.Username
	}
	if req.Status != nil {
		grpcReq.Status = req.Status
	}

	resp, err := c.client.UpdateProfile(callCtx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC UpdateProfile failed: %w", err)
	}

	return &UserResponse{User: protoToUserDTO(resp.User)}, nil
}

func (c *identityClient) SearchUsers(ctx context.Context, req *SearchUsersRequest) (*SearchUsersResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.SearchUsers(callCtx, &identityv1.SearchUsersRequest{
		Query:  req.Query,
		Limit:  int32(req.Limit),
		Offset: int32(req.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC SearchUsers failed: %w", err)
	}

	users := make([]UserDTO, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, protoToUserDTO(u))
	}

	return &SearchUsersResponse{Users: users}, nil
}

func (c *identityClient) SendContact(ctx context.Context, req *SendContactRequest) (*ContactRequestResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.SendContact(callCtx, &identityv1.SendContactRequest{
		UserId:    req.UserID.String(),
		ContactId: req.ContactID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC SendContact failed: %w", err)
	}

	return &ContactRequestResponse{
		RequestID: resp.RequestId,
		Status:    resp.Status,
	}, nil
}

func (c *identityClient) AcceptContact(ctx context.Context, req *AcceptContactRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.AcceptContact(callCtx, &identityv1.AcceptContactRequest{
		UserId:        req.UserID.String(),
		RequestFromId: req.RequestFromID.String(),
	})
	return err
}

func (c *identityClient) RejectContact(ctx context.Context, req *RejectContactRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.RejectContact(callCtx, &identityv1.RejectContactRequest{
		UserId:        req.UserID.String(),
		RequestFromId: req.RequestFromID.String(),
	})
	return err
}

func (c *identityClient) ListPendingRequests(ctx context.Context, req *ListPendingRequestsRequest) (*ListPendingRequestsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ListPendingRequests(callCtx, &identityv1.ListPendingRequestsRequest{
		UserId: req.UserID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC ListPendingRequests failed: %w", err)
	}

	requests := make([]UserDTO, 0, len(resp.Requests))
	for _, r := range resp.Requests {
		requests = append(requests, protoToUserDTO(r))
	}

	return &ListPendingRequestsResponse{Requests: requests}, nil
}

func (c *identityClient) DeleteContact(ctx context.Context, req *DeleteContactRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.DeleteContact(callCtx, &identityv1.DeleteContactRequest{
		UserId:    req.UserID.String(),
		ContactId: req.ContactID.String(),
	})
	return err
}

func (c *identityClient) ListContacts(ctx context.Context, req *ListContactsRequest) (*ListContactsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ListContacts(callCtx, &identityv1.ListContactsRequest{
		UserId: req.UserID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC ListContacts failed: %w", err)
	}

	contacts := make([]UserDTO, 0, len(resp.Contacts))
	for _, c := range resp.Contacts {
		contacts = append(contacts, protoToUserDTO(c))
	}

	return &ListContactsResponse{Contacts: contacts}, nil
}

func (c *identityClient) GetUploadAvatarURL(ctx context.Context, req *GetUploadAvatarURLRequest) (*GetUploadURLResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetUploadAvatarURL(callCtx, &identityv1.GetUploadAvatarURLRequest{
		OwnerId:  req.OwnerID.String(),
		FileName: req.FileName,
		MimeType: req.MimeType,
		FileSize: req.FileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetUploadAvatarURL failed: %w", err)
	}

	return &GetUploadURLResponse{
		FileID:    uuid.MustParse(resp.FileId),
		UploadURL: resp.UploadUrl,
		ObjectKey: resp.ObjectKey,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	}, nil
}

func (c *identityClient) ConfirmAvatarUpload(ctx context.Context, req *ConfirmUploadRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.ConfirmAvatarUpload(callCtx, &identityv1.ConfirmUploadRequest{
		FileId: req.FileID.String(),
	})
	return err
}

func (c *identityClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// protoToUserDTO — вспомогательная функция конвертации
func protoToUserDTO(u *identityv1.UserDTO) UserDTO {
	return UserDTO{
		ID:        uuid.MustParse(u.Id),
		Username:  u.Username,
		Email:     u.Email,
		AvatarURL: u.AvatarUrl,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.AsTime(),
	}
}
