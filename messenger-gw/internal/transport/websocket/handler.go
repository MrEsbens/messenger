package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MrEsbens/messenger-gw/internal/auth"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/chats"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/file"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/identity"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/messages"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/servers"
	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types"
	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/commands"
	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/data"
	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/responses"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WSHandler — обработчик WebSocket-соединений
type WSHandler struct {
	hub            *Hub
	upgrader       *websocket.Upgrader
	jwtSecret      []byte
	messagesClient messages.MessagesClientInterface
	identityClient identity.IdentityClientInterface
	chatsClient    chats.ChatsClientInterface
	serversClient  servers.ServersClientInterface
	fileClient     file.FileClientInterface
	log            *log.Logger
}

// NewWSHandler создаёт обработчик
func NewWSHandler(
	h *Hub,
	upgrader *websocket.Upgrader,
	jwtSecret []byte,
	messagesClient messages.MessagesClientInterface,
	identityClient identity.IdentityClientInterface,
	chatsClient chats.ChatsClientInterface,
	serversClient servers.ServersClientInterface,
	fileClient file.FileClientInterface,
) *WSHandler {
	return &WSHandler{
		hub:            h,
		upgrader:       upgrader,
		jwtSecret:      jwtSecret,
		messagesClient: messagesClient,
		identityClient: identityClient,
		chatsClient:    chatsClient,
		serversClient:  serversClient,
		fileClient:     fileClient,
		log:            log.Default(),
	}
}

// ServeHTTP — точка входа для WebSocket upgrade
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.ExtractUserID(r, h.jwtSecret)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Printf("WS upgrade failed: %v", err)
		return
	}

	client := NewClient(conn, userID, h.hub)
	h.hub.Register <- client

	go client.WritePump()
	client.ReadPump()
}

// ─── Роутер ───────────────────────────────────────────────────────────────

// HandleMessage — главный диспетчер команд
func (h *WSHandler) HandleMessage(c *Client, raw []byte) {
	var cmd commands.Command
	if err := json.Unmarshal(raw, &cmd); err != nil {
		h.sendError(c, "invalid_json", err.Error(), "")
		return
	}

	switch cmd.Cmd {
	case "ping":
		h.sendSuccess(c, "pong", &responses.PongPayload{}, cmd.RequestID)
	case "subscribe":
		h.handleSubscribe(c, cmd.Payload, cmd.RequestID)
	default:
		if h.routeIdentity(c, cmd) {
			return
		}
		if h.routeMessages(c, cmd) {
			return
		}
		if h.routeChats(c, cmd) {
			return
		}
		if h.routeServers(c, cmd) {
			return
		}
		if h.routeFile(c, cmd) {
			return
		}

		h.sendError(c, "unknown_command", fmt.Sprintf("command '%s' not supported", cmd.Cmd), cmd.RequestID)
	}
}

// routeIdentity — маршрутизатор команд Identity Service
func (h *WSHandler) routeIdentity(c *Client, cmd commands.Command) bool {
	switch cmd.Cmd {
	case "register":
		h.handleRegister(c, cmd.Payload, cmd.RequestID)
	case "login":
		h.handleLogin(c, cmd.Payload, cmd.RequestID)
	case "get_profile":
		h.handleGetProfile(c, cmd.Payload, cmd.RequestID)
	case "update_profile":
		h.handleUpdateProfile(c, cmd.Payload, cmd.RequestID)
	case "search_users":
		h.handleSearchUsers(c, cmd.Payload, cmd.RequestID)
	case "send_contact":
		h.handleSendContact(c, cmd.Payload, cmd.RequestID)
	case "accept_contact":
		h.handleAcceptContact(c, cmd.Payload, cmd.RequestID)
	case "reject_contact":
		h.handleRejectContact(c, cmd.Payload, cmd.RequestID)
	case "list_pending_requests":
		h.handleListPendingRequests(c, cmd.Payload, cmd.RequestID)
	case "list_contacts":
		h.handleListContacts(c, cmd.Payload, cmd.RequestID)
	case "delete_contact":
		h.handleDeleteContact(c, cmd.Payload, cmd.RequestID)
	case "get_upload_avatar_url":
		h.handleGetUploadAvatarURL(c, cmd.Payload, cmd.RequestID)
	case "confirm_avatar_upload":
		h.handleConfirmAvatarUpload(c, cmd.Payload, cmd.RequestID)
	default:
		return false
	}
	return true
}

// routeMessages — маршрутизатор команд Messages Service
func (h *WSHandler) routeMessages(c *Client, cmd commands.Command) bool {
	switch cmd.Cmd {
	case "send_message":
		h.handleSendMessage(c, cmd.Payload, cmd.RequestID)
	case "mark_read":
		h.handleMarkRead(c, cmd.Payload, cmd.RequestID)
	case "edit_message":
		h.handleEditMessage(c, cmd.Payload, cmd.RequestID)
	case "delete_message":
		h.handleDeleteMessage(c, cmd.Payload, cmd.RequestID)
	case "get_messages":
		h.handleGetMessages(c, cmd.Payload, cmd.RequestID)
	case "get_message_statuses":
		h.handleGetMessageStatuses(c, cmd.Payload, cmd.RequestID)
	case "get_message_with_attachments":
		h.handleGetMessageWithAttachments(c, cmd.Payload, cmd.RequestID)
	default:
		return false
	}
	return true
}

func (h *WSHandler) routeChats(c *Client, cmd commands.Command) bool {
	switch cmd.Cmd {
	case "create_chat":
		h.handleCreateChat(c, cmd.Payload, cmd.RequestID)
	case "get_chat":
		h.handleGetChat(c, cmd.Payload, cmd.RequestID)
	case "update_chat":
		h.handleUpdateChat(c, cmd.Payload, cmd.RequestID)
	case "delete_chat":
		h.handleDeleteChat(c, cmd.Payload, cmd.RequestID)
	case "list_chats":
		h.handleListChats(c, cmd.Payload, cmd.RequestID)
	case "add_member":
		h.handleAddMember(c, cmd.Payload, cmd.RequestID)
	case "remove_member":
		h.handleRemoveMember(c, cmd.Payload, cmd.RequestID)
	case "update_member_role":
		h.handleUpdateMemberRole(c, cmd.Payload, cmd.RequestID)
	case "list_chat_members":
		h.handleListChatMembers(c, cmd.Payload, cmd.RequestID)
	case "create_invite":
		h.handleCreateInvite(c, cmd.Payload, cmd.RequestID)
	case "join_by_invite":
		h.handleJoinByInvite(c, cmd.Payload, cmd.RequestID)
	default:
		return false
	}
	return true
}

func (h *WSHandler) routeServers(c *Client, cmd commands.Command) bool {
	switch cmd.Cmd {
	// ─── Server CRUD ─────────────────────────────────────
	case "create_server":
		h.handleCreateServer(c, cmd.Payload, cmd.RequestID)
	case "get_server":
		h.handleGetServer(c, cmd.Payload, cmd.RequestID)
	case "update_server":
		h.handleUpdateServer(c, cmd.Payload, cmd.RequestID)
	case "delete_server":
		h.handleDeleteServer(c, cmd.Payload, cmd.RequestID)
	case "list_servers":
		h.handleListServers(c, cmd.Payload, cmd.RequestID)

	// ─── Configs ─────────────────────────────────────────
	case "get_server_config":
		h.handleGetServerConfig(c, cmd.Payload, cmd.RequestID)
	case "update_server_config":
		h.handleUpdateServerConfig(c, cmd.Payload, cmd.RequestID)
	case "get_moderation_config":
		h.handleGetModerationConfig(c, cmd.Payload, cmd.RequestID)
	case "update_moderation_config":
		h.handleUpdateModerationConfig(c, cmd.Payload, cmd.RequestID)

	// ─── Members ─────────────────────────────────────────
	case "add_server_member":
		h.handleAddServerMember(c, cmd.Payload, cmd.RequestID)
	case "remove_server_member":
		h.handleRemoveServerMember(c, cmd.Payload, cmd.RequestID)
	case "ban_member":
		h.handleBanMember(c, cmd.Payload, cmd.RequestID)
	case "unban_member":
		h.handleUnbanMember(c, cmd.Payload, cmd.RequestID)
	case "mute_member":
		h.handleMuteMember(c, cmd.Payload, cmd.RequestID)
	case "unmute_member":
		h.handleUnmuteMember(c, cmd.Payload, cmd.RequestID)

	// ─── Roles ───────────────────────────────────────────
	case "create_role":
		h.handleCreateRole(c, cmd.Payload, cmd.RequestID)
	case "update_role":
		h.handleUpdateRole(c, cmd.Payload, cmd.RequestID)
	case "delete_role":
		h.handleDeleteRole(c, cmd.Payload, cmd.RequestID)
	case "assign_role":
		h.handleAssignRole(c, cmd.Payload, cmd.RequestID)
	case "remove_role":
		h.handleRemoveRole(c, cmd.Payload, cmd.RequestID)
	case "get_member_roles":
		h.handleGetMemberRoles(c, cmd.Payload, cmd.RequestID)

	// ─── Server Chats ────────────────────────────────────
	case "create_server_chat":
		h.handleCreateServerChat(c, cmd.Payload, cmd.RequestID)
	case "list_server_chats":
		h.handleListServerChats(c, cmd.Payload, cmd.RequestID)
	case "delete_server_chat":
		h.handleDeleteServerChat(c, cmd.Payload, cmd.RequestID)

	default:
		return false
	}
	return true
}

func (h *WSHandler) routeFile(c *Client, cmd commands.Command) bool {
	switch cmd.Cmd {
	case "get_upload_url":
		h.handleGetUploadURL(c, cmd.Payload, cmd.RequestID)
	case "confirm_upload":
		h.handleConfirmUpload(c, cmd.Payload, cmd.RequestID)
	case "get_download_url":
		h.handleGetDownloadURL(c, cmd.Payload, cmd.RequestID)
	case "delete_file":
		h.handleDeleteFile(c, cmd.Payload, cmd.RequestID)
	case "get_files_by_owner":
		h.handleGetFilesByOwner(c, cmd.Payload, cmd.RequestID)
	default:
		return false
	}
	return true
}

// ─── Утилиты ─────────────────────────────────────────────────────────────

func (h *WSHandler) requireAuth(c *Client, requestID string) bool {
	if c.UserID == "" {
		h.sendError(c, "unauthorized", "authentication required", requestID)
		return false
	}
	return true
}

func (h *WSHandler) sendError(c *Client, code, msg, reqID string) {
	_ = c.SendJSON(&responses.Response{
		Type:      "error",
		Payload:   &responses.ErrorPayload{Code: code, Message: msg},
		RequestID: reqID,
	})
}

func (h *WSHandler) sendSuccess(c *Client, eventType string, payload interface{}, reqID string) {
	_ = c.SendJSON(&responses.Response{
		Type:      eventType,
		Payload:   payload,
		RequestID: reqID,
	})
}

// ─── Identity Service Handlers ────────────────────────────────────────────

// ─── Identity Service Handlers (AUTH) ─────────────────────────────────────

// handleRegister — регистрация нового пользователя
func (h *WSHandler) handleRegister(c *Client, payload json.RawMessage, requestID string) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse register", requestID)
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.sendError(c, "validation_error", "username, email and password are required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.identityClient.Register(ctx, &identity.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		// Проверяем на конфликт (пользователь уже существует)
		if strings.Contains(err.Error(), "already exists") {
			h.sendError(c, "user_exists", "user with this username or email already exists", requestID)
			return
		}
		h.sendError(c, "registration_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	// 🔥 Возвращаем токен + данные пользователя
	h.sendSuccess(c, "auth", map[string]interface{}{
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
		"user": map[string]interface{}{
			"id":         resp.User.ID.String(),
			"username":   resp.User.Username,
			"email":      resp.User.Email,
			"avatar_url": resp.User.AvatarURL,
			"status":     resp.User.Status,
			"created_at": types.FormatTime(resp.User.CreatedAt),
		},
	}, requestID)
}

// handleLogin — аутентификация пользователя
func (h *WSHandler) handleLogin(c *Client, payload json.RawMessage, requestID string) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse login", requestID)
		return
	}
	if req.Email == "" || req.Password == "" {
		h.sendError(c, "validation_error", "email and password are required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.identityClient.Login(ctx, &identity.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.sendError(c, "auth_failed", "invalid credentials", requestID)
		return
	}

	// 🔥 Возвращаем токен + данные пользователя
	h.sendSuccess(c, "auth", map[string]interface{}{
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
		"user": map[string]interface{}{
			"id":         resp.User.ID.String(),
			"username":   resp.User.Username,
			"email":      resp.User.Email,
			"avatar_url": resp.User.AvatarURL,
			"status":     resp.User.Status,
			"created_at": types.FormatTime(resp.User.CreatedAt),
		},
	}, requestID)
}

func (h *WSHandler) handleGetProfile(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.identityClient.GetMe(ctx, &identity.GetMeRequest{UserID: uuid.MustParse(c.UserID)})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get profile: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "profile", map[string]interface{}{
		"id":         resp.User.ID.String(),
		"username":   resp.User.Username,
		"email":      resp.User.Email,
		"avatar_url": resp.User.AvatarURL,
		"status":     resp.User.Status,
		"created_at": types.FormatTime(resp.User.CreatedAt),
	}, requestID)
}

func (h *WSHandler) handleUpdateProfile(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		Username *string `json:"username"`
		Status   *string `json:"status"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse update_profile", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.identityClient.UpdateProfile(ctx, &identity.UpdateProfileRequest{
		UserID:   uuid.MustParse(c.UserID),
		Username: req.Username,
		Status:   req.Status,
	})
	if err != nil {
		h.sendError(c, "update_failed", fmt.Sprintf("failed to update profile: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "profile_updated", map[string]interface{}{
		"username": resp.User.Username,
		"status":   resp.User.Status,
	}, requestID)
}

func (h *WSHandler) handleSearchUsers(c *Client, payload json.RawMessage, requestID string) {
	var req struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.Query == "" {
		h.sendError(c, "invalid_payload", "query is required", requestID)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.identityClient.SearchUsers(ctx, &identity.SearchUsersRequest{
		Query:  req.Query,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		h.sendError(c, "search_failed", fmt.Sprintf("search failed: %v", err), requestID)
		return
	}

	users := make([]map[string]string, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, map[string]string{
			"id":         u.ID.String(),
			"username":   u.Username,
			"avatar_url": u.AvatarURL,
			"status":     u.Status,
		})
	}

	h.sendSuccess(c, "search_results", map[string]interface{}{
		"query": req.Query,
		"users": users,
	}, requestID)
}

func (h *WSHandler) handleSendContact(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ContactID string `json:"contact_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ContactID == "" {
		h.sendError(c, "invalid_payload", "contact_id is required", requestID)
		return
	}

	contactID, err := types.ParseUUID(req.ContactID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.identityClient.SendContact(ctx, &identity.SendContactRequest{
		UserID:    uuid.MustParse(c.UserID),
		ContactID: contactID,
	})
	if err != nil {
		h.sendError(c, "send_request_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "contact_request_sent", map[string]interface{}{
		"request_id": resp.RequestID,
		"status":     resp.Status,
	}, requestID)
}

func (h *WSHandler) handleAcceptContact(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		RequestFromID string `json:"request_from_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.RequestFromID == "" {
		h.sendError(c, "invalid_payload", "request_from_id is required", requestID)
		return
	}

	requestFromID, err := types.ParseUUID(req.RequestFromID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.identityClient.AcceptContact(ctx, &identity.AcceptContactRequest{
		UserID:        uuid.MustParse(c.UserID),
		RequestFromID: requestFromID,
	}); err != nil {
		h.sendError(c, "accept_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "contact_accepted", map[string]string{"request_from_id": req.RequestFromID}, requestID)
}

func (h *WSHandler) handleRejectContact(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		RequestFromID string `json:"request_from_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.RequestFromID == "" {
		h.sendError(c, "invalid_payload", "request_from_id is required", requestID)
		return
	}

	requestFromID, err := types.ParseUUID(req.RequestFromID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.identityClient.RejectContact(ctx, &identity.RejectContactRequest{
		UserID:        uuid.MustParse(c.UserID),
		RequestFromID: requestFromID,
	}); err != nil {
		h.sendError(c, "reject_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "contact_rejected", map[string]string{"request_from_id": req.RequestFromID}, requestID)
}

func (h *WSHandler) handleListPendingRequests(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.identityClient.ListPendingRequests(ctx, &identity.ListPendingRequestsRequest{
		UserID: uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get pending requests: %v", err), requestID)
		return
	}

	requests := make([]map[string]string, 0, len(resp.Requests))
	for _, r := range resp.Requests {
		requests = append(requests, map[string]string{
			"id":       r.ID.String(),
			"username": r.Username,
			"status":   r.Status,
		})
	}

	h.sendSuccess(c, "pending_requests", map[string]interface{}{"requests": requests}, requestID)
}

func (h *WSHandler) handleListContacts(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.identityClient.ListContacts(ctx, &identity.ListContactsRequest{
		UserID: uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get contacts: %v", err), requestID)
		return
	}

	contacts := make([]map[string]string, 0, len(resp.Contacts))
	for _, c := range resp.Contacts {
		contacts = append(contacts, map[string]string{
			"id":         c.ID.String(),
			"username":   c.Username,
			"avatar_url": c.AvatarURL,
			"status":     c.Status,
		})
	}

	h.sendSuccess(c, "contacts_list", map[string]interface{}{"contacts": contacts}, requestID)
}

func (h *WSHandler) handleDeleteContact(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ContactID string `json:"contact_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ContactID == "" {
		h.sendError(c, "invalid_payload", "contact_id is required", requestID)
		return
	}

	contactID, err := types.ParseUUID(req.ContactID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.identityClient.DeleteContact(ctx, &identity.DeleteContactRequest{
		UserID:    uuid.MustParse(c.UserID),
		ContactID: contactID,
	}); err != nil {
		h.sendError(c, "delete_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "contact_deleted", map[string]string{"contact_id": req.ContactID}, requestID)
}

func (h *WSHandler) handleGetUploadAvatarURL(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		FileName string `json:"file_name"`
		MimeType string `json:"mime_type"`
		FileSize int64  `json:"file_size"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.FileName == "" || req.MimeType == "" {
		h.sendError(c, "invalid_payload", "file_name and mime_type are required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.identityClient.GetUploadAvatarURL(ctx, &identity.GetUploadAvatarURLRequest{
		OwnerID:  uuid.MustParse(c.UserID),
		FileName: req.FileName,
		MimeType: req.MimeType,
		FileSize: req.FileSize,
	})
	if err != nil {
		h.sendError(c, "upload_url_failed", fmt.Sprintf("failed to get upload url: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "upload_url_ready", map[string]interface{}{
		"file_id":    resp.FileID.String(),
		"upload_url": resp.UploadURL,
		"object_key": resp.ObjectKey,
		"expires_at": types.FormatTime(resp.ExpiresAt),
	}, requestID)
}

func (h *WSHandler) handleConfirmAvatarUpload(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.FileID == "" {
		h.sendError(c, "invalid_payload", "file_id is required", requestID)
		return
	}

	fileID, err := types.ParseUUID(req.FileID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.identityClient.ConfirmAvatarUpload(ctx, &identity.ConfirmUploadRequest{FileID: fileID}); err != nil {
		h.sendError(c, "confirm_failed", fmt.Sprintf("failed to confirm upload: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "avatar_upload_confirmed", map[string]string{"file_id": req.FileID}, requestID)
}

// ─── Messages Service Handlers ───────────────────────────────────────────

func (h *WSHandler) handleSendMessage(c *Client, payload json.RawMessage, requestID string) {
	var req commands.SendMessagePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse send_message", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}
	if !h.requireAuth(c, requestID) {
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_chat_id", err.Error(), requestID)
		return
	}
	var replyToID *uuid.UUID
	if req.ReplyToID != "" {
		id, err := types.ParseUUID(req.ReplyToID)
		if err != nil {
			h.sendError(c, "invalid_reply_to_id", err.Error(), requestID)
			return
		}
		replyToID = &id
	}
	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil {
		h.sendError(c, "invalid_server_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.messagesClient.SendMessage(ctx, &messages.SendMessageRequest{
		SenderID:    uuid.MustParse(c.UserID),
		ChatID:      chatID,
		ServerID:    serverID,
		Content:     req.Content,
		ContentType: req.ContentType,
		ReplyToID:   replyToID,
	})
	if err != nil {
		h.sendError(c, "send_failed", fmt.Sprintf("failed to send: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "message_sent", &responses.MessageSentPayload{
		MessageID: resp.MessageID.String(),
		CreatedAt: types.FormatTime(resp.CreatedAt),
	}, requestID)
}

func (h *WSHandler) handleGetMessages(c *Client, payload json.RawMessage, requestID string) {
	var req commands.GetMessagesPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse get_messages", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_chat_id", err.Error(), requestID)
		return
	}

	var after *time.Time
	if req.After != nil {
		t, err := time.Parse(time.RFC3339, *req.After)
		if err == nil {
			after = &t
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := h.messagesClient.GetMessages(ctx, &messages.GetMessagesRequest{
		ChatID: chatID,
		Limit:  req.Limit,
		Offset: req.Offset,
		After:  after,
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get messages: %v", err), requestID)
		return
	}

	payloads := make([]*data.MessagePayload, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		payloads = append(payloads, messages.MessageToTransportPayload(m))
	}

	h.sendSuccess(c, "messages_list", &responses.GetMessagesResponsePayload{
		Messages: payloads,
		HasMore:  resp.HasMore,
	}, requestID)
}

func (h *WSHandler) handleGetMessageWithAttachments(c *Client, payload json.RawMessage, requestID string) {
	var req commands.GetMessageWithAttachmentsPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse get_message_with_attachments", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}

	messageID, err := types.ParseUUID(req.MessageID)
	if err != nil {
		h.sendError(c, "invalid_message_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.messagesClient.GetMessageWithAttachments(ctx, messageID)
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get message: %v", err), requestID)
		return
	}

	msgPayload := messages.MessageToTransportPayload(resp.Message)
	attachments := make([]*data.AttachmentPreview, 0, len(resp.Attachments))
	for _, a := range resp.Attachments {
		attachments = append(attachments, messages.AttachmentToTransportPreview(a))
	}

	h.sendSuccess(c, "message_with_attachments", &responses.GetMessageWithAttachmentsResponsePayload{
		Message:     msgPayload,
		Attachments: attachments,
	}, requestID)
}

func (h *WSHandler) handleEditMessage(c *Client, payload json.RawMessage, requestID string) {
	var req commands.EditMessagePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse edit_message", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}
	if !h.requireAuth(c, requestID) {
		return
	}

	messageID, err := types.ParseUUID(req.MessageID)
	if err != nil {
		h.sendError(c, "invalid_message_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.messagesClient.EditMessage(ctx, &messages.EditMessageRequest{
		MessageID:  messageID,
		EditorID:   uuid.MustParse(c.UserID),
		NewContent: req.NewContent,
	}); err != nil {
		h.sendError(c, "edit_failed", fmt.Sprintf("failed to edit: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "message_edited", &responses.MessageEditedPayload{
		ID:         req.MessageID,
		ChatID:     "",
		NewContent: req.NewContent,
		UpdatedAt:  types.FormatTime(time.Now()),
		EditorID:   c.UserID,
	}, requestID)
}

func (h *WSHandler) handleDeleteMessage(c *Client, payload json.RawMessage, requestID string) {
	var req commands.DeleteMessagePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse delete_message", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}
	if !h.requireAuth(c, requestID) {
		return
	}

	messageID, err := types.ParseUUID(req.MessageID)
	if err != nil {
		h.sendError(c, "invalid_message_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.messagesClient.DeleteMessage(ctx, &messages.DeleteMessageRequest{
		MessageID: messageID,
		DeleterID: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "delete_failed", fmt.Sprintf("failed to delete: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "message_deleted", &responses.MessageDeletedPayload{
		MessageID: req.MessageID,
		DeletedAt: types.FormatTime(time.Now()),
	}, requestID)
}

func (h *WSHandler) handleMarkRead(c *Client, payload json.RawMessage, requestID string) {
	var req commands.MarkReadPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse mark_read", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}
	if !h.requireAuth(c, requestID) {
		return
	}

	messageID, err := types.ParseUUID(req.MessageID)
	if err != nil {
		h.sendError(c, "invalid_message_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.messagesClient.UpdateMessageStatus(ctx, &messages.UpdateStatusRequest{
		MessageID: messageID,
		UserID:    uuid.MustParse(c.UserID),
		Status:    "read",
	}); err != nil {
		h.sendError(c, "status_update_failed", fmt.Sprintf("failed to update: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "status_updated", map[string]string{
		"message_id": req.MessageID,
		"status":     "read",
		"updated_at": types.FormatTime(time.Now()),
	}, requestID)
}

func (h *WSHandler) handleGetMessageStatuses(c *Client, payload json.RawMessage, requestID string) {
	var req commands.GetMessageStatusesPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse get_message_statuses", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}

	messageID, err := types.ParseUUID(req.MessageID)
	if err != nil {
		h.sendError(c, "invalid_message_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.messagesClient.GetMessageStatuses(ctx, &messages.GetMessageStatusesRequest{MessageID: messageID})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get statuses: %v", err), requestID)
		return
	}

	statuses := make([]responses.MessageStatusPayload, 0, len(resp.Statuses))
	for _, s := range resp.Statuses {
		statuses = append(statuses, responses.MessageStatusPayload{
			UserID:    s.UserID.String(),
			Status:    s.Status,
			UpdatedAt: types.FormatTime(s.UpdatedAt),
		})
	}

	h.sendSuccess(c, "message_statuses", &responses.GetMessageStatusesResponsePayload{Statuses: statuses}, requestID)
}

// ─── Chats Service Handlers ─────────────────────────────────────────────

func (h *WSHandler) handleCreateChat(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		Type             string   `json:"type"` // "private", "group", "server"
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		AvatarFileID     string   `json:"avatar_file_id"`
		InitialMemberIDs []string `json:"initial_member_ids"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse create_chat", requestID)
		return
	}
	if req.Type == "" {
		h.sendError(c, "validation_error", "chat type is required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Конвертация начальных участников
	initialMembers := make([]uuid.UUID, 0, len(req.InitialMemberIDs))
	for _, id := range req.InitialMemberIDs {
		uid, err := types.ParseUUID(id)
		if err != nil {
			h.sendError(c, "invalid_member_id", fmt.Sprintf("invalid uuid: %s", id), requestID)
			return
		}
		initialMembers = append(initialMembers, uid)
	}

	var avatarFileID *uuid.UUID
	if req.AvatarFileID != "" {
		id, err := types.ParseUUID(req.AvatarFileID)
		if err != nil {
			h.sendError(c, "invalid_avatar_file_id", err.Error(), requestID)
			return
		}
		avatarFileID = &id
	}

	resp, err := h.chatsClient.CreateChat(ctx, &chats.CreateChatRequest{
		Type:             chats.ChatType(req.Type),
		Name:             req.Name,
		Description:      req.Description,
		AvatarFileID:     avatarFileID,
		InitialMemberIDs: initialMembers,
		CreatedBy:        uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "create_failed", fmt.Sprintf("failed to create chat: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "chat_created", map[string]interface{}{
		"chat_id":    resp.ChatID.String(),
		"created_at": types.FormatTime(resp.CreatedAt),
	}, requestID)
}

func (h *WSHandler) handleGetChat(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID string `json:"chat_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" {
		h.sendError(c, "invalid_payload", "chat_id is required", requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.chatsClient.GetChat(ctx, &chats.GetChatRequest{
		ChatID:      chatID,
		RequesterID: uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get chat: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "chat", map[string]interface{}{
		"id":                   resp.ID.String(),
		"type":                 resp.Type,
		"name":                 resp.Name,
		"description":          resp.Description,
		"avatar_url":           resp.AvatarURL,
		"has_avatar":           resp.HasAvatar,
		"created_by":           resp.CreatedBy.String(),
		"member_count":         resp.MemberCount,
		"last_message_preview": resp.LastMessagePreview,
		"created_at":           types.FormatTime(resp.CreatedAt),
		"updated_at":           types.FormatTime(resp.UpdatedAt),
		"is_deleted":           resp.IsDeleted,
	}, requestID)
}

func (h *WSHandler) handleUpdateChat(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID       string  `json:"chat_id"`
		Name         *string `json:"name"`
		Description  *string `json:"description"`
		AvatarFileID *string `json:"avatar_file_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" {
		h.sendError(c, "invalid_payload", "chat_id is required", requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	var avatarFileID *uuid.UUID
	if req.AvatarFileID != nil && *req.AvatarFileID != "" {
		id, err := types.ParseUUID(*req.AvatarFileID)
		if err != nil {
			h.sendError(c, "invalid_avatar_file_id", err.Error(), requestID)
			return
		}
		avatarFileID = &id
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.chatsClient.UpdateChat(ctx, &chats.UpdateChatRequest{
		ChatID:       chatID,
		UpdaterID:    uuid.MustParse(c.UserID),
		Name:         req.Name,
		Description:  req.Description,
		AvatarFileID: avatarFileID,
	}); err != nil {
		h.sendError(c, "update_failed", fmt.Sprintf("failed to update chat: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "chat_updated", map[string]string{"chat_id": req.ChatID}, requestID)
}

func (h *WSHandler) handleDeleteChat(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID string `json:"chat_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" {
		h.sendError(c, "invalid_payload", "chat_id is required", requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.chatsClient.DeleteChat(ctx, &chats.DeleteChatRequest{
		ChatID:    chatID,
		DeleterID: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "delete_failed", fmt.Sprintf("failed to delete chat: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "chat_deleted", map[string]string{"chat_id": req.ChatID}, requestID)
}

func (h *WSHandler) handleListChats(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		TypeFilter *string `json:"type_filter"` // "private", "group", "server"
		Limit      int     `json:"limit"`
		Offset     int     `json:"offset"`
		AfterID    *string `json:"after_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse list_chats", requestID)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var typeFilter *chats.ChatType
	if req.TypeFilter != nil && *req.TypeFilter != "" {
		t := chats.ChatType(*req.TypeFilter)
		typeFilter = &t
	}

	resp, err := h.chatsClient.ListUserChats(ctx, &chats.ListUserChatsRequest{
		UserID:     uuid.MustParse(c.UserID),
		TypeFilter: typeFilter,
		Limit:      req.Limit,
		Offset:     req.Offset,
		AfterID:    req.AfterID,
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get chats: %v", err), requestID)
		return
	}

	chatsPayload := make([]map[string]interface{}, 0, len(resp.Chats))
	for _, chat := range resp.Chats {
		chatsPayload = append(chatsPayload, map[string]interface{}{
			"id":                   chat.ID.String(),
			"type":                 chat.Type,
			"name":                 chat.Name,
			"avatar_url":           chat.AvatarURL,
			"member_count":         chat.MemberCount,
			"last_message_preview": chat.LastMessagePreview,
			"updated_at":           types.FormatTime(chat.UpdatedAt),
			"unread_count":         0, // TODO: получить из messages-service
		})
	}

	h.sendSuccess(c, "chats_list", map[string]interface{}{
		"chats":    chatsPayload,
		"has_more": resp.HasMore,
	}, requestID)
}

func (h *WSHandler) handleAddMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID string `json:"chat_id"`
		UserID string `json:"user_id"`
		Role   string `json:"role"` // "member", "admin"
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "chat_id and user_id are required", requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_chat_id", err.Error(), requestID)
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil {
		h.sendError(c, "invalid_user_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.chatsClient.AddMember(ctx, &chats.AddMemberRequest{
		ChatID:  chatID,
		UserID:  userID,
		AddedBy: uuid.MustParse(c.UserID),
		Role:    chats.MembershipRole(req.Role),
	}); err != nil {
		h.sendError(c, "add_member_failed", fmt.Sprintf("failed to add member: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_added", map[string]string{
		"chat_id": req.ChatID,
		"user_id": req.UserID,
	}, requestID)
}

func (h *WSHandler) handleRemoveMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID string `json:"chat_id"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "chat_id and user_id are required", requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_chat_id", err.Error(), requestID)
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil {
		h.sendError(c, "invalid_user_id", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.chatsClient.RemoveMember(ctx, &chats.RemoveMemberRequest{
		ChatID:    chatID,
		UserID:    userID,
		RemovedBy: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "remove_member_failed", fmt.Sprintf("failed to remove member: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_removed", map[string]string{
		"chat_id": req.ChatID,
		"user_id": req.UserID,
	}, requestID)
}

func (h *WSHandler) handleUpdateMemberRole(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID       string `json:"chat_id"`
		TargetUserID string `json:"target_user_id"`
		NewRole      string `json:"new_role"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" || req.TargetUserID == "" || req.NewRole == "" {
		h.sendError(c, "invalid_payload", "chat_id, target_user_id and new_role are required", requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil { /* error */
		return
	}
	targetUserID, err := types.ParseUUID(req.TargetUserID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.chatsClient.UpdateMemberRole(ctx, &chats.UpdateMemberRoleRequest{
		ChatID:       chatID,
		TargetUserID: targetUserID,
		UpdatedBy:    uuid.MustParse(c.UserID),
		NewRole:      chats.MembershipRole(req.NewRole),
	}); err != nil {
		h.sendError(c, "update_role_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "role_updated", map[string]string{
		"chat_id": req.ChatID,
		"user_id": req.TargetUserID,
		"role":    req.NewRole,
	}, requestID)
}

func (h *WSHandler) handleListChatMembers(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID     string  `json:"chat_id"`
		RoleFilter *string `json:"role_filter"`
		Limit      int     `json:"limit"`
		Offset     int     `json:"offset"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" {
		h.sendError(c, "invalid_payload", "chat_id is required", requestID)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_chat_id", err.Error(), requestID)
		return
	}

	var roleFilter *chats.MembershipRole
	if req.RoleFilter != nil && *req.RoleFilter != "" {
		r := chats.MembershipRole(*req.RoleFilter)
		roleFilter = &r
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.chatsClient.ListChatMembers(ctx, &chats.ListChatMembersRequest{
		ChatID:      chatID,
		RequesterID: uuid.MustParse(c.UserID),
		RoleFilter:  roleFilter,
		Limit:       req.Limit,
		Offset:      req.Offset,
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get members: %v", err), requestID)
		return
	}

	members := make([]map[string]interface{}, 0, len(resp.Members))
	for _, m := range resp.Members {
		members = append(members, map[string]interface{}{
			"user_id":           m.UserID.String(),
			"role":              m.Role,
			"joined_at":         types.FormatTime(m.JoinedAt),
			"notification_mode": m.NotificationMode,
			"last_activity_at":  types.FormatTime(m.LastActivityAt),
		})
	}

	h.sendSuccess(c, "chat_members", map[string]interface{}{
		"members":  members,
		"has_more": resp.HasMore,
	}, requestID)
}

func (h *WSHandler) handleCreateInvite(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ChatID    string  `json:"chat_id"`
		ExpiresAt *string `json:"expires_at"` // RFC3339
		MaxUses   *int32  `json:"max_uses"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ChatID == "" {
		h.sendError(c, "invalid_payload", "chat_id is required", requestID)
		return
	}

	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil {
		h.sendError(c, "invalid_chat_id", err.Error(), requestID)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.chatsClient.CreateInvite(ctx, &chats.CreateInviteRequest{
		ChatID:    chatID,
		CreatedBy: uuid.MustParse(c.UserID),
		ExpiresAt: expiresAt,
		MaxUses:   req.MaxUses,
	})
	if err != nil {
		h.sendError(c, "create_invite_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "invite_created", map[string]interface{}{
		"invite_token": resp.InviteToken,
		"expires_at":   types.FormatTime(resp.ExpiresAt),
	}, requestID)
}

func (h *WSHandler) handleJoinByInvite(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		InviteToken string `json:"invite_token"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.InviteToken == "" {
		h.sendError(c, "invalid_payload", "invite_token is required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.chatsClient.JoinByInvite(ctx, &chats.JoinByInviteRequest{
		InviteToken: req.InviteToken,
		UserID:      uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "join_failed", fmt.Sprintf("failed to join: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "joined_chat", map[string]string{"invite_token": req.InviteToken}, requestID)
}

// ─── Servers Service Handlers ───────────────────────────────────────────

// ─── Server CRUD ────────────────────────────────────────────────────────

func (h *WSHandler) handleCreateServer(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.Name == "" {
		h.sendError(c, "invalid_payload", "name is required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.serversClient.CreateServer(ctx, &servers.CreateServerRequest{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "create_failed", fmt.Sprintf("failed to create server: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server_created", map[string]interface{}{
		"server_id":  resp.ServerID.String(),
		"created_at": types.FormatTime(resp.CreatedAt),
	}, requestID)
}

func (h *WSHandler) handleGetServer(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.serversClient.GetServer(ctx, &servers.GetServerRequest{
		ServerID:    serverID,
		RequesterID: uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get server: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server", map[string]interface{}{
		"id":          resp.ID.String(),
		"name":        resp.Name,
		"description": resp.Description,
		"owner_id":    resp.OwnerID.String(),
		"created_at":  types.FormatTime(resp.CreatedAt),
		"updated_at":  types.FormatTime(resp.UpdatedAt),
		"is_deleted":  resp.IsDeleted,
	}, requestID)
}

func (h *WSHandler) handleUpdateServer(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID    string  `json:"server_id"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.UpdateServer(ctx, &servers.UpdateServerRequest{
		ServerID:    serverID,
		UpdaterID:   uuid.MustParse(c.UserID),
		Name:        req.Name,
		Description: req.Description,
	}); err != nil {
		h.sendError(c, "update_failed", fmt.Sprintf("failed to update server: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server_updated", map[string]string{"server_id": req.ServerID}, requestID)
}

func (h *WSHandler) handleDeleteServer(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.DeleteServer(ctx, &servers.DeleteServerRequest{
		ServerID:  serverID,
		DeleterID: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "delete_failed", fmt.Sprintf("failed to delete server: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server_deleted", map[string]string{"server_id": req.ServerID}, requestID)
}

func (h *WSHandler) handleListServers(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse list_servers", requestID)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.serversClient.ListUserServers(ctx, &servers.ListUserServersRequest{
		UserID: uuid.MustParse(c.UserID),
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get servers: %v", err), requestID)
		return
	}

	serversPayload := make([]map[string]interface{}, 0, len(resp.Servers))
	for _, s := range resp.Servers {
		serversPayload = append(serversPayload, map[string]interface{}{
			"id":          s.ID.String(),
			"name":        s.Name,
			"description": s.Description,
			"owner_id":    s.OwnerID.String(),
			"created_at":  types.FormatTime(s.CreatedAt),
			"updated_at":  types.FormatTime(s.UpdatedAt),
		})
	}

	h.sendSuccess(c, "servers_list", map[string]interface{}{
		"servers": serversPayload,
	}, requestID)
}

// ─── Configs ────────────────────────────────────────────────────────────

func (h *WSHandler) handleGetServerConfig(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.serversClient.GetServerConfig(ctx, &servers.GetServerConfigRequest{ServerID: serverID})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get config: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server_config", map[string]interface{}{
		"server_id":                 resp.ServerID.String(),
		"max_members":               resp.MaxMembers,
		"max_channels":              resp.MaxChannels,
		"default_notification_mode": resp.DefaultNotificationMode,
		"moderation_enabled":        resp.ModerationEnabled,
	}, requestID)
}

func (h *WSHandler) handleUpdateServerConfig(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID                string  `json:"server_id"`
		MaxMembers              *int32  `json:"max_members"`
		MaxChannels             *int32  `json:"max_channels"`
		DefaultNotificationMode *string `json:"default_notification_mode"`
		ModerationEnabled       *bool   `json:"moderation_enabled"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.UpdateServerConfig(ctx, &servers.UpdateServerConfigRequest{
		ServerID:                uuid.MustParse(req.ServerID),
		UpdaterID:               uuid.MustParse(c.UserID),
		MaxMembers:              req.MaxMembers,
		MaxChannels:             req.MaxChannels,
		DefaultNotificationMode: req.DefaultNotificationMode,
		ModerationEnabled:       req.ModerationEnabled,
	}); err != nil {
		h.sendError(c, "update_failed", fmt.Sprintf("failed to update config: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server_config_updated", map[string]string{"server_id": req.ServerID}, requestID)
}

func (h *WSHandler) handleGetModerationConfig(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.serversClient.GetModerationConfig(ctx, &servers.GetModerationConfigRequest{ServerID: serverID})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get moderation config: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "moderation_config", map[string]interface{}{
		"server_id":                 resp.ServerID.String(),
		"profanity_filter_action":   resp.ProfanityFilterAction,
		"toxicity_filter_action":    resp.ToxicityFilterAction,
		"nsfw_text_filter_action":   resp.NsfwTextFilterAction,
		"hate_speech_filter_action": resp.HateSpeechFilterAction,
	}, requestID)
}

func (h *WSHandler) handleUpdateModerationConfig(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID               string  `json:"server_id"`
		ProfanityFilterAction  *string `json:"profanity_filter_action"`
		ToxicityFilterAction   *string `json:"toxicity_filter_action"`
		NsfwTextFilterAction   *string `json:"nsfw_text_filter_action"`
		HateSpeechFilterAction *string `json:"hate_speech_filter_action"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.UpdateModerationConfig(ctx, &servers.UpdateModerationConfigRequest{
		ServerID:               uuid.MustParse(req.ServerID),
		UpdaterID:              uuid.MustParse(c.UserID),
		ProfanityFilterAction:  req.ProfanityFilterAction,
		ToxicityFilterAction:   req.ToxicityFilterAction,
		NsfwTextFilterAction:   req.NsfwTextFilterAction,
		HateSpeechFilterAction: req.HateSpeechFilterAction,
	}); err != nil {
		h.sendError(c, "update_failed", fmt.Sprintf("failed to update moderation config: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "moderation_config_updated", map[string]string{"server_id": req.ServerID}, requestID)
}

// ─── Members ────────────────────────────────────────────────────────────

func (h *WSHandler) handleAddServerMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		UserID   string `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "server_id and user_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.AddMember(ctx, &servers.AddMemberRequest{
		ServerID: serverID,
		UserID:   userID,
		AddedBy:  uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "add_member_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_added", map[string]string{
		"server_id": req.ServerID,
		"user_id":   req.UserID,
	}, requestID)
}

func (h *WSHandler) handleRemoveServerMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		UserID   string `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "server_id and user_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.RemoveMember(ctx, &servers.RemoveMemberRequest{
		ServerID:  serverID,
		UserID:    userID,
		RemovedBy: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "remove_member_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_removed", map[string]string{
		"server_id": req.ServerID,
		"user_id":   req.UserID,
	}, requestID)
}

func (h *WSHandler) handleBanMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		UserID   string `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "server_id and user_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.BanMember(ctx, &servers.BanMemberRequest{
		ServerID: serverID,
		UserID:   userID,
		BannedBy: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "ban_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_banned", map[string]string{
		"server_id": req.ServerID,
		"user_id":   req.UserID,
	}, requestID)
}

func (h *WSHandler) handleUnbanMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		UserID   string `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "server_id and user_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.UnbanMember(ctx, &servers.UnbanMemberRequest{
		ServerID:   serverID,
		UserID:     userID,
		UnbannedBy: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "unban_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_unbanned", map[string]string{
		"server_id": req.ServerID,
		"user_id":   req.UserID,
	}, requestID)
}

func (h *WSHandler) handleMuteMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID      string `json:"server_id"`
		UserID        string `json:"user_id"`
		DurationHours int64  `json:"duration_hours"` // 0 = навсегда
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "server_id and user_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.MuteMember(ctx, &servers.MuteMemberRequest{
		ServerID:      serverID,
		UserID:        userID,
		MutedBy:       uuid.MustParse(c.UserID),
		DurationHours: req.DurationHours,
	}); err != nil {
		h.sendError(c, "mute_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_muted", map[string]interface{}{
		"server_id":      req.ServerID,
		"user_id":        req.UserID,
		"duration_hours": req.DurationHours,
	}, requestID)
}

func (h *WSHandler) handleUnmuteMember(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		UserID   string `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.UserID == "" {
		h.sendError(c, "invalid_payload", "server_id and user_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	userID, err := types.ParseUUID(req.UserID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.UnmuteMember(ctx, &servers.UnmuteMemberRequest{
		ServerID:  serverID,
		UserID:    userID,
		UnmutedBy: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "unmute_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "member_unmuted", map[string]string{
		"server_id": req.ServerID,
		"user_id":   req.UserID,
	}, requestID)
}

// ─── Roles ──────────────────────────────────────────────────────────────

func (h *WSHandler) handleCreateRole(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID    string   `json:"server_id"`
		Name        string   `json:"name"`
		Color       string   `json:"color"`
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.Name == "" {
		h.sendError(c, "invalid_payload", "server_id and name are required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.serversClient.CreateRole(ctx, &servers.CreateRoleRequest{
		ServerID:  uuid.MustParse(req.ServerID),
		Name:      req.Name,
		CreatorID: uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "create_role_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "role_created", map[string]interface{}{
		"role_id":     resp.ID.String(),
		"name":        resp.Name,
		"color":       resp.Color,
		"permissions": resp.Permissions,
		"position":    resp.Position,
	}, requestID)
}

func (h *WSHandler) handleUpdateRole(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID    string   `json:"server_id"`
		RoleID      string   `json:"role_id"`
		Name        *string  `json:"name"`
		Color       *string  `json:"color"`
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.RoleID == "" {
		h.sendError(c, "invalid_payload", "server_id and role_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	roleID, err := types.ParseUUID(req.RoleID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.UpdateRole(ctx, &servers.UpdateRoleRequest{
		ServerID:    serverID,
		RoleID:      roleID,
		UpdaterID:   uuid.MustParse(c.UserID),
		Name:        req.Name,
		Color:       req.Color,
		Permissions: req.Permissions,
	}); err != nil {
		h.sendError(c, "update_role_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "role_updated", map[string]string{
		"server_id": req.ServerID,
		"role_id":   req.RoleID,
	}, requestID)
}

func (h *WSHandler) handleDeleteRole(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		RoleID   string `json:"role_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.RoleID == "" {
		h.sendError(c, "invalid_payload", "server_id and role_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	roleID, err := types.ParseUUID(req.RoleID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.DeleteRole(ctx, &servers.DeleteRoleRequest{
		ServerID:  serverID,
		RoleID:    roleID,
		DeleterID: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "delete_role_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "role_deleted", map[string]string{
		"server_id": req.ServerID,
		"role_id":   req.RoleID,
	}, requestID)
}

func (h *WSHandler) handleAssignRole(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		MemberID string `json:"member_id"`
		RoleID   string `json:"role_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.MemberID == "" || req.RoleID == "" {
		h.sendError(c, "invalid_payload", "server_id, member_id and role_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	memberID, err := types.ParseUUID(req.MemberID)
	if err != nil { /* error */
		return
	}
	roleID, err := types.ParseUUID(req.RoleID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.AssignRole(ctx, &servers.AssignRoleRequest{
		ServerID:   serverID,
		MemberID:   memberID,
		RoleID:     roleID,
		AssignedBy: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "assign_role_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "role_assigned", map[string]string{
		"server_id": req.ServerID,
		"member_id": req.MemberID,
		"role_id":   req.RoleID,
	}, requestID)
}

func (h *WSHandler) handleRemoveRole(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		MemberID string `json:"member_id"`
		RoleID   string `json:"role_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.MemberID == "" || req.RoleID == "" {
		h.sendError(c, "invalid_payload", "server_id, member_id and role_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	memberID, err := types.ParseUUID(req.MemberID)
	if err != nil { /* error */
		return
	}
	roleID, err := types.ParseUUID(req.RoleID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.RemoveRole(ctx, &servers.RemoveRoleRequest{
		ServerID:  serverID,
		MemberID:  memberID,
		RoleID:    roleID,
		RemovedBy: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "remove_role_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "role_removed", map[string]string{
		"server_id": req.ServerID,
		"member_id": req.MemberID,
		"role_id":   req.RoleID,
	}, requestID)
}

func (h *WSHandler) handleGetMemberRoles(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		MemberID string `json:"member_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.MemberID == "" {
		h.sendError(c, "invalid_payload", "member_id is required", requestID)
		return
	}

	memberID, err := types.ParseUUID(req.MemberID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.serversClient.GetMemberRoles(ctx, &servers.GetMemberRolesRequest{MemberID: memberID})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get roles: %v", err), requestID)
		return
	}

	roles := make([]map[string]interface{}, 0, len(resp.Roles))
	for _, r := range resp.Roles {
		roles = append(roles, map[string]interface{}{
			"id":          r.ID.String(),
			"name":        r.Name,
			"color":       r.Color,
			"permissions": r.Permissions,
			"position":    r.Position,
			"is_default":  r.IsDefault,
		})
	}

	h.sendSuccess(c, "member_roles", map[string]interface{}{
		"member_id": req.MemberID,
		"roles":     roles,
	}, requestID)
}

// ─── Server Chats ───────────────────────────────────────────────────────

func (h *WSHandler) handleCreateServerChat(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.Name == "" {
		h.sendError(c, "invalid_payload", "server_id and name are required", requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.serversClient.CreateServerChat(ctx, &servers.CreateServerChatRequest{
		ServerID:  uuid.MustParse(req.ServerID),
		Name:      req.Name,
		CreatedBy: uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "create_chat_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server_chat_created", map[string]interface{}{
		"chat_id":    resp.ChatID.String(),
		"server_id":  resp.ServerID.String(),
		"name":       resp.Name,
		"position":   resp.Position,
		"created_at": types.FormatTime(resp.CreatedAt),
	}, requestID)
}

func (h *WSHandler) handleListServerChats(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" {
		h.sendError(c, "invalid_payload", "server_id is required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.serversClient.ListServerChats(ctx, &servers.ListServerChatsRequest{
		ServerID:    serverID,
		RequesterID: uuid.MustParse(c.UserID),
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get chats: %v", err), requestID)
		return
	}

	chats := make([]map[string]interface{}, 0, len(resp.Chats))
	for _, c := range resp.Chats {
		chats = append(chats, map[string]interface{}{
			"chat_id":    c.ChatID.String(),
			"server_id":  c.ServerID.String(),
			"name":       c.Name,
			"position":   c.Position,
			"created_at": types.FormatTime(c.CreatedAt),
		})
	}

	h.sendSuccess(c, "server_chats", map[string]interface{}{
		"server_id": req.ServerID,
		"chats":     chats,
	}, requestID)
}

func (h *WSHandler) handleDeleteServerChat(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
		ChatID   string `json:"chat_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.ServerID == "" || req.ChatID == "" {
		h.sendError(c, "invalid_payload", "server_id and chat_id are required", requestID)
		return
	}

	serverID, err := types.ParseUUID(req.ServerID)
	if err != nil { /* error */
		return
	}
	chatID, err := types.ParseUUID(req.ChatID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.serversClient.DeleteServerChat(ctx, &servers.DeleteServerChatRequest{
		ServerID:  serverID,
		ChatID:    chatID,
		DeleterID: uuid.MustParse(c.UserID),
	}); err != nil {
		h.sendError(c, "delete_chat_failed", fmt.Sprintf("failed: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "server_chat_deleted", map[string]string{
		"server_id": req.ServerID,
		"chat_id":   req.ChatID,
	}, requestID)
}

// ─── File Service Handlers ──────────────────────────────────────────────

func (h *WSHandler) handleGetUploadURL(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		OwnerID    string `json:"owner_id"`    // Чей файл (user_id, chat_id, server_id)
		OwnerType  string `json:"owner_type"`  // user, chat, server, message
		BucketType string `json:"bucket_type"` // avatars, attachments, server_icons
		FileName   string `json:"file_name"`
		MimeType   string `json:"mime_type"`
		FileSize   int64  `json:"file_size"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse get_upload_url", requestID)
		return
	}

	// Валидация обязательных полей
	if req.OwnerID == "" || req.OwnerType == "" || req.BucketType == "" || req.FileName == "" || req.MimeType == "" {
		h.sendError(c, "validation_error", "owner_id, owner_type, bucket_type, file_name and mime_type are required", requestID)
		return
	}
	if req.FileSize <= 0 {
		h.sendError(c, "validation_error", "file_size must be positive", requestID)
		return
	}

	ownerID, err := types.ParseUUID(req.OwnerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.fileClient.GetUploadURL(ctx, &file.GetUploadURLRequest{
		OwnerID:    ownerID,
		OwnerType:  req.OwnerType,
		BucketType: req.BucketType,
		FileName:   req.FileName,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
	})
	if err != nil {
		h.sendError(c, "upload_url_failed", fmt.Sprintf("failed to get url: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "upload_url_ready", map[string]interface{}{
		"file_id":    resp.FileID.String(),
		"upload_url": resp.UploadURL,
		"object_key": resp.ObjectKey,
		"expires_at": types.FormatTime(resp.ExpiresAt),
	}, requestID)
}

func (h *WSHandler) handleConfirmUpload(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.FileID == "" {
		h.sendError(c, "invalid_payload", "file_id is required", requestID)
		return
	}

	fileID, err := types.ParseUUID(req.FileID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.fileClient.ConfirmUpload(ctx, &file.ConfirmUploadRequest{FileID: fileID}); err != nil {
		h.sendError(c, "confirm_failed", fmt.Sprintf("failed to confirm upload: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "upload_confirmed", map[string]string{"file_id": req.FileID}, requestID)
}

func (h *WSHandler) handleGetDownloadURL(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		FileID        string `json:"file_id"`
		ExpirySeconds int64  `json:"expiry_seconds"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.FileID == "" {
		h.sendError(c, "invalid_payload", "file_id is required", requestID)
		return
	}

	fileID, err := types.ParseUUID(req.FileID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.fileClient.GetDownloadURL(ctx, &file.GetDownloadURLRequest{
		FileID:        fileID,
		ExpirySeconds: req.ExpirySeconds,
	})
	if err != nil {
		h.sendError(c, "download_url_failed", fmt.Sprintf("failed to get url: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "download_url_ready", map[string]interface{}{
		"download_url": resp.DownloadURL,
		"expires_at":   types.FormatTime(resp.ExpiresAt),
	}, requestID)
}

func (h *WSHandler) handleDeleteFile(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		FileID  string `json:"file_id"`
		OwnerID string `json:"owner_id"` // Нужен для проверки прав на стороне сервиса
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.FileID == "" || req.OwnerID == "" {
		h.sendError(c, "invalid_payload", "file_id and owner_id are required", requestID)
		return
	}

	fileID, err := types.ParseUUID(req.FileID)
	if err != nil { /* error */
		return
	}
	ownerID, err := types.ParseUUID(req.OwnerID)
	if err != nil { /* error */
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.fileClient.DeleteFile(ctx, &file.DeleteFileRequest{
		FileID:  fileID,
		OwnerID: ownerID,
	}); err != nil {
		h.sendError(c, "delete_failed", fmt.Sprintf("failed to delete file: %v", err), requestID)
		return
	}

	h.sendSuccess(c, "file_deleted", map[string]string{"file_id": req.FileID}, requestID)
}

func (h *WSHandler) handleGetFilesByOwner(c *Client, payload json.RawMessage, requestID string) {
	if !h.requireAuth(c, requestID) {
		return
	}

	var req struct {
		OwnerID    string `json:"owner_id"`
		OwnerType  string `json:"owner_type"`
		BucketType string `json:"bucket_type"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.OwnerID == "" || req.OwnerType == "" || req.BucketType == "" {
		h.sendError(c, "invalid_payload", "owner_id, owner_type and bucket_type are required", requestID)
		return
	}

	ownerID, err := types.ParseUUID(req.OwnerID)
	if err != nil {
		h.sendError(c, "invalid_uuid", err.Error(), requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.fileClient.GetFilesByOwner(ctx, &file.GetFilesByOwnerRequest{
		OwnerID:    ownerID,
		OwnerType:  req.OwnerType,
		BucketType: req.BucketType,
	})
	if err != nil {
		h.sendError(c, "fetch_failed", fmt.Sprintf("failed to get files: %v", err), requestID)
		return
	}

	files := make([]map[string]interface{}, 0, len(resp.Files))
	for _, f := range resp.Files {
		files = append(files, map[string]interface{}{
			"file_id":    f.FileID.String(),
			"public_url": f.PublicURL,
			"mime_type":  f.MimeType,
			"file_size":  f.FileSize,
			"file_name":  f.FileName,
		})
	}

	h.sendSuccess(c, "files_list", map[string]interface{}{
		"files": files,
	}, requestID)
}

// ─── Hub / Subscribe ─────────────────────────────────────────────────────

func (h *WSHandler) handleSubscribe(c *Client, payload json.RawMessage, requestID string) {
	var req commands.SubscribePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid_payload", "failed to parse subscribe", requestID)
		return
	}
	if err := req.IsValid(); err != nil {
		h.sendError(c, "validation_error", err.Error(), requestID)
		return
	}

	targetID := req.ChatID
	if targetID == "" {
		targetID = req.ServerChannelID
	}

	h.hub.Subscribe(c, targetID)

	h.sendSuccess(c, "subscribed", &responses.SubscribedPayload{
		ChatID:          req.ChatID,
		ServerChannelID: req.ServerChannelID,
		Timestamp:       types.FormatTime(time.Now()),
	}, requestID)
}

func (h *WSHandler) SendToSubscribers(targetID string, eventType string, payload interface{}) {
	h.hub.SendToSubscribers(targetID, &responses.Response{
		Type:    eventType,
		Payload: payload,
	})
}
