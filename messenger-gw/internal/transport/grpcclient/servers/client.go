package servers

import (
	"context"
	"fmt"
	"strings"
	"time"

	serversv1 "github.com/MrEsbens/messenger-servers-service/api/servers/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServersClientInterface interface {
	CreateServer(ctx context.Context, req *CreateServerRequest) (*CreateServerResponse, error)
	GetServer(ctx context.Context, req *GetServerRequest) (*ServerDTO, error)
	UpdateServer(ctx context.Context, req *UpdateServerRequest) error
	DeleteServer(ctx context.Context, req *DeleteServerRequest) error
	ListUserServers(ctx context.Context, req *ListUserServersRequest) (*ListUserServersResponse, error)
	GetServerConfig(ctx context.Context, req *GetServerConfigRequest) (*ServerConfigDTO, error)
	UpdateServerConfig(ctx context.Context, req *UpdateServerConfigRequest) error
	GetModerationConfig(ctx context.Context, req *GetModerationConfigRequest) (*ModerationConfigDTO, error)
	UpdateModerationConfig(ctx context.Context, req *UpdateModerationConfigRequest) error
	AddMember(ctx context.Context, req *AddMemberRequest) error
	RemoveMember(ctx context.Context, req *RemoveMemberRequest) error
	GetMember(ctx context.Context, req *GetMemberRequest) (*MemberDTO, error)
	ListMembers(ctx context.Context, req *ListMembersRequest) (*ListMembersResponse, error)
	BanMember(ctx context.Context, req *BanMemberRequest) error
	UnbanMember(ctx context.Context, req *UnbanMemberRequest) error
	MuteMember(ctx context.Context, req *MuteMemberRequest) error
	UnmuteMember(ctx context.Context, req *UnmuteMemberRequest) error
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*RoleDTO, error)
	UpdateRole(ctx context.Context, req *UpdateRoleRequest) error
	DeleteRole(ctx context.Context, req *DeleteRoleRequest) error
	AssignRole(ctx context.Context, req *AssignRoleRequest) error
	RemoveRole(ctx context.Context, req *RemoveRoleRequest) error
	GetMemberRoles(ctx context.Context, req *GetMemberRolesRequest) (*GetMemberRolesResponse, error)
	CreateServerChat(ctx context.Context, req *CreateServerChatRequest) (*ServerChatDTO, error)
	ListServerChats(ctx context.Context, req *ListServerChatsRequest) (*ListServerChatsResponse, error)
	DeleteServerChat(ctx context.Context, req *DeleteServerChatRequest) error
	Close() error
}

type serversClient struct {
	client  serversv1.ServersServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewServersClient(endpoint string, timeout time.Duration) (ServersClientInterface, error) {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "grpc://")

	if endpoint == "" {
		return nil, fmt.Errorf("empty servers service endpoint")
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithReturnConnectionError(),
	}

	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to servers service: %w", err)
	}

	return &serversClient{
		client:  serversv1.NewServersServiceClient(conn),
		conn:    conn,
		timeout: timeout,
	}, nil
}

func (c *serversClient) CreateServer(ctx context.Context, req *CreateServerRequest) (*CreateServerResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &serversv1.CreateServerRequest{
		Name:    req.Name,
		OwnerId: req.OwnerID.String(),
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	resp, err := c.client.CreateServer(callCtx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC CreateServer failed: %w", err)
	}

	return &CreateServerResponse{
		ServerID:  uuid.MustParse(resp.ServerId),
		CreatedAt: resp.CreatedAt.AsTime(),
	}, nil
}

func (c *serversClient) GetServer(ctx context.Context, req *GetServerRequest) (*ServerDTO, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetServer(callCtx, &serversv1.GetServerRequest{
		ServerId:    req.ServerID.String(),
		RequesterId: req.RequesterID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetServer failed: %w", err)
	}

	dto := protoToServerDTO(resp)
	return &dto, nil
}

func (c *serversClient) UpdateServer(ctx context.Context, req *UpdateServerRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &serversv1.UpdateServerRequest{
		ServerId:  req.ServerID.String(),
		UpdaterId: req.UpdaterID.String(),
	}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	_, err := c.client.UpdateServer(callCtx, grpcReq)
	return err
}

func (c *serversClient) DeleteServer(ctx context.Context, req *DeleteServerRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.DeleteServer(callCtx, &serversv1.DeleteServerRequest{
		ServerId:  req.ServerID.String(),
		DeleterId: req.DeleterID.String(),
	})
	return err
}

func (c *serversClient) ListUserServers(ctx context.Context, req *ListUserServersRequest) (*ListUserServersResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ListUserServers(callCtx, &serversv1.ListUserServersRequest{
		UserId: req.UserID.String(),
		Limit:  int32(req.Limit),
		Offset: int32(req.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC ListUserServers failed: %w", err)
	}

	servers := make([]ServerDTO, 0, len(resp.Servers))
	for _, s := range resp.Servers {
		dto := protoToServerDTO(s)
		servers = append(servers, dto)
	}

	return &ListUserServersResponse{Servers: servers}, nil
}

func (c *serversClient) GetServerConfig(ctx context.Context, req *GetServerConfigRequest) (*ServerConfigDTO, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetServerConfig(callCtx, &serversv1.GetServerConfigRequest{
		ServerId: req.ServerID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetServerConfig failed: %w", err)
	}

	return &ServerConfigDTO{
		ServerID:                uuid.MustParse(resp.ServerId),
		MaxMembers:              resp.MaxMembers,
		MaxChannels:             resp.MaxChannels,
		DefaultNotificationMode: resp.DefaultNotificationMode,
		ModerationEnabled:       resp.ModerationEnabled,
	}, nil
}

func (c *serversClient) UpdateServerConfig(ctx context.Context, req *UpdateServerConfigRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &serversv1.UpdateServerConfigRequest{
		ServerId:  req.ServerID.String(),
		UpdaterId: req.UpdaterID.String(),
	}
	if req.MaxMembers != nil {
		grpcReq.MaxMembers = req.MaxMembers
	}
	if req.MaxChannels != nil {
		grpcReq.MaxChannels = req.MaxChannels
	}
	if req.DefaultNotificationMode != nil {
		grpcReq.DefaultNotificationMode = req.DefaultNotificationMode
	}
	if req.ModerationEnabled != nil {
		grpcReq.ModerationEnabled = req.ModerationEnabled
	}

	_, err := c.client.UpdateServerConfig(callCtx, grpcReq)
	return err
}

func (c *serversClient) GetModerationConfig(ctx context.Context, req *GetModerationConfigRequest) (*ModerationConfigDTO, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetModerationConfig(callCtx, &serversv1.GetModerationConfigRequest{
		ServerId: req.ServerID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetModerationConfig failed: %w", err)
	}

	return &ModerationConfigDTO{
		ServerID:               uuid.MustParse(resp.ServerId),
		ProfanityFilterAction:  resp.ProfanityFilterAction,
		ToxicityFilterAction:   resp.ToxicityFilterAction,
		NsfwTextFilterAction:   resp.NsfwTextFilterAction,
		HateSpeechFilterAction: resp.HateSpeechFilterAction,
	}, nil
}

func (c *serversClient) UpdateModerationConfig(ctx context.Context, req *UpdateModerationConfigRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &serversv1.UpdateModerationConfigRequest{
		ServerId:  req.ServerID.String(),
		UpdaterId: req.UpdaterID.String(),
	}
	if req.ProfanityFilterAction != nil {
		grpcReq.ProfanityFilterAction = req.ProfanityFilterAction
	}
	if req.ToxicityFilterAction != nil {
		grpcReq.ToxicityFilterAction = req.ToxicityFilterAction
	}
	if req.NsfwTextFilterAction != nil {
		grpcReq.NsfwTextFilterAction = req.NsfwTextFilterAction
	}
	if req.HateSpeechFilterAction != nil {
		grpcReq.HateSpeechFilterAction = req.HateSpeechFilterAction
	}

	_, err := c.client.UpdateModerationConfig(callCtx, grpcReq)
	return err
}

func (c *serversClient) AddMember(ctx context.Context, req *AddMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.AddMember(callCtx, &serversv1.AddMemberRequest{
		ServerId: req.ServerID.String(),
		UserId:   req.UserID.String(),
		AddedBy:  req.AddedBy.String(),
	})
	return err
}

func (c *serversClient) RemoveMember(ctx context.Context, req *RemoveMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.RemoveMember(callCtx, &serversv1.RemoveMemberRequest{
		ServerId:  req.ServerID.String(),
		UserId:    req.UserID.String(),
		RemovedBy: req.RemovedBy.String(),
	})
	return err
}

func (c *serversClient) GetMember(ctx context.Context, req *GetMemberRequest) (*MemberDTO, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetMember(callCtx, &serversv1.GetMemberRequest{
		ServerId: req.ServerID.String(),
		UserId:   req.UserID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetMember failed: %w", err)
	}

	dto := protoToMemberDTO(resp)
	return &dto, nil
}

func (c *serversClient) ListMembers(ctx context.Context, req *ListMembersRequest) (*ListMembersResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ListMembers(callCtx, &serversv1.ListMembersRequest{
		ServerId: req.ServerID.String(),
		Limit:    int32(req.Limit),
		Offset:   int32(req.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC ListMembers failed: %w", err)
	}

	members := make([]MemberDTO, 0, len(resp.Members))
	for _, m := range resp.Members {
		dto := protoToMemberDTO(m)
		members = append(members, dto)
	}

	return &ListMembersResponse{Members: members}, nil
}

func (c *serversClient) BanMember(ctx context.Context, req *BanMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.BanMember(callCtx, &serversv1.BanMemberRequest{
		ServerId: req.ServerID.String(),
		UserId:   req.UserID.String(),
		BannedBy: req.BannedBy.String(),
	})
	return err
}

func (c *serversClient) UnbanMember(ctx context.Context, req *UnbanMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.UnbanMember(callCtx, &serversv1.UnbanMemberRequest{
		ServerId:   req.ServerID.String(),
		UserId:     req.UserID.String(),
		UnbannedBy: req.UnbannedBy.String(),
	})
	return err
}

func (c *serversClient) MuteMember(ctx context.Context, req *MuteMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.MuteMember(callCtx, &serversv1.MuteMemberRequest{
		ServerId:      req.ServerID.String(),
		UserId:        req.UserID.String(),
		MutedBy:       req.MutedBy.String(),
		DurationHours: req.DurationHours,
	})
	return err
}

func (c *serversClient) UnmuteMember(ctx context.Context, req *UnmuteMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.UnmuteMember(callCtx, &serversv1.UnmuteMemberRequest{
		ServerId:  req.ServerID.String(),
		UserId:    req.UserID.String(),
		UnmutedBy: req.UnmutedBy.String(),
	})
	return err
}

func (c *serversClient) CreateRole(ctx context.Context, req *CreateRoleRequest) (*RoleDTO, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.CreateRole(callCtx, &serversv1.CreateRoleRequest{
		ServerId:  req.ServerID.String(),
		Name:      req.Name,
		CreatorId: req.CreatorID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC CreateRole failed: %w", err)
	}

	dto := protoToRoleDTO(resp)
	return &dto, nil
}

func (c *serversClient) UpdateRole(ctx context.Context, req *UpdateRoleRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &serversv1.UpdateRoleRequest{
		ServerId:  req.ServerID.String(),
		RoleId:    req.RoleID.String(),
		UpdaterId: req.UpdaterID.String(),
	}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}
	grpcReq.Permissions = req.Permissions

	_, err := c.client.UpdateRole(callCtx, grpcReq)
	return err
}

func (c *serversClient) DeleteRole(ctx context.Context, req *DeleteRoleRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.DeleteRole(callCtx, &serversv1.DeleteRoleRequest{
		ServerId:  req.ServerID.String(),
		RoleId:    req.RoleID.String(),
		DeleterId: req.DeleterID.String(),
	})
	return err
}

func (c *serversClient) AssignRole(ctx context.Context, req *AssignRoleRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.AssignRole(callCtx, &serversv1.AssignRoleRequest{
		ServerId:   req.ServerID.String(),
		MemberId:   req.MemberID.String(),
		RoleId:     req.RoleID.String(),
		AssignedBy: req.AssignedBy.String(),
	})
	return err
}

func (c *serversClient) RemoveRole(ctx context.Context, req *RemoveRoleRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.RemoveRole(callCtx, &serversv1.RemoveRoleRequest{
		ServerId:  req.ServerID.String(),
		MemberId:  req.MemberID.String(),
		RoleId:    req.RoleID.String(),
		RemovedBy: req.RemovedBy.String(),
	})
	return err
}

func (c *serversClient) GetMemberRoles(ctx context.Context, req *GetMemberRolesRequest) (*GetMemberRolesResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetMemberRoles(callCtx, &serversv1.GetMemberRolesRequest{
		MemberId: req.MemberID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetMemberRoles failed: %w", err)
	}

	roles := make([]RoleDTO, 0, len(resp.Roles))
	for _, r := range resp.Roles {
		dto := protoToRoleDTO(r)
		roles = append(roles, dto)
	}

	return &GetMemberRolesResponse{Roles: roles}, nil
}

func (c *serversClient) CreateServerChat(ctx context.Context, req *CreateServerChatRequest) (*ServerChatDTO, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.CreateServerChat(callCtx, &serversv1.CreateServerChatRequest{
		ServerId:  req.ServerID.String(),
		Name:      req.Name,
		CreatedBy: req.CreatedBy.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC CreateServerChat failed: %w", err)
	}

	return &ServerChatDTO{
		ChatID:    uuid.MustParse(resp.ChatId),
		ServerID:  uuid.MustParse(resp.ServerId),
		Name:      resp.Name,
		Position:  resp.Position,
		CreatedAt: resp.CreatedAt.AsTime(),
	}, nil
}

func (c *serversClient) ListServerChats(ctx context.Context, req *ListServerChatsRequest) (*ListServerChatsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ListServerChats(callCtx, &serversv1.ListServerChatsRequest{
		ServerId:    req.ServerID.String(),
		RequesterId: req.RequesterID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC ListServerChats failed: %w", err)
	}

	chats := make([]ServerChatDTO, 0, len(resp.Chats))
	for _, chat := range resp.Chats {
		chats = append(chats, ServerChatDTO{
			ChatID:    uuid.MustParse(chat.ChatId),
			ServerID:  uuid.MustParse(chat.ServerId),
			Name:      chat.Name,
			Position:  chat.Position,
			CreatedAt: chat.CreatedAt.AsTime(),
		})
	}

	return &ListServerChatsResponse{Chats: chats}, nil
}

func (c *serversClient) DeleteServerChat(ctx context.Context, req *DeleteServerChatRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.DeleteServerChat(callCtx, &serversv1.DeleteServerChatRequest{
		ServerId:  req.ServerID.String(),
		ChatId:    req.ChatID.String(),
		DeleterId: req.DeleterID.String(),
	})
	return err
}

func (c *serversClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ─── Вспомогательные функции конвертации ─────────────────────

func protoToServerDTO(s *serversv1.ServerDTO) ServerDTO {
	var description *string
	if s.Description != nil && *s.Description != "" {
		description = s.Description
	}
	return ServerDTO{
		ID:          uuid.MustParse(s.Id),
		Name:        s.Name,
		Description: description,
		OwnerID:     uuid.MustParse(s.OwnerId),
		CreatedAt:   s.CreatedAt.AsTime(),
		UpdatedAt:   s.UpdatedAt.AsTime(),
		IsDeleted:   s.IsDeleted,
	}
}

func protoToMemberDTO(m *serversv1.MemberDTO) MemberDTO {
	roleIDs := make([]uuid.UUID, 0, len(m.RoleIds))
	for _, id := range m.RoleIds {
		roleIDs = append(roleIDs, uuid.MustParse(id))
	}
	var mutedUntil *time.Time
	if m.MutedUntil != nil {
		t := m.MutedUntil.AsTime()
		mutedUntil = &t
	}
	return MemberDTO{
		UserID:     uuid.MustParse(m.UserId),
		ServerID:   uuid.MustParse(m.ServerId),
		RoleIDs:    roleIDs,
		JoinedAt:   m.JoinedAt.AsTime(),
		IsMuted:    m.IsMuted,
		IsBanned:   m.IsBanned,
		MutedUntil: mutedUntil,
	}
}

func protoToRoleDTO(r *serversv1.RoleDTO) RoleDTO {
	return RoleDTO{
		ID:          uuid.MustParse(r.Id),
		ServerID:    uuid.MustParse(r.ServerId),
		Name:        r.Name,
		Color:       r.Color,
		Permissions: r.Permissions,
		Position:    r.Position,
		IsDefault:   r.IsDefault,
	}
}
