package domain

import (
	"errors"
)

var (
	ErrPrivateChatLimit      = errors.New("private chat must have exactly 2 members")
	ErrGroupChatLimit        = errors.New("group chat member limit exceeded")
	ErrServerChatLimit       = errors.New("server chat member limit exceeded")
	ErrCannotChangeOwnerRole = errors.New("cannot change owner's role")
	ErrOwnerCannotLeave      = errors.New("owner cannot leave the chat")
	ErrPrivateChatNoAdmin    = errors.New("private chats cannot have admin role")
	ErrGroupChatNoAdmin      = errors.New("group chats cannot have admin role")
)

func ValidateChatMemberLimit(chatType ChatType, currentCount, newCount int) error {
	switch chatType {
	case ChatTypePrivate:
		if newCount < MinPrivateChatMembers || newCount > MaxPrivateChatMembers {
			return ErrPrivateChatLimit
		}
	case ChatTypeGroup:
		if newCount > MaxGroupChatMembers {
			return ErrGroupChatLimit
		}
	case ChatTypeServer:
		if newCount > MaxServerChatMembers {
			return ErrServerChatLimit
		}
	}
	return nil
}

func CanUpdateChat(chatType ChatType, role MembershipRole, isOwner bool) bool {
	if isOwner {
		return true
	}
	if chatType == ChatTypePrivate {
		return false
	}
	return role == RoleAdmin
}

func CanAddMember(chatType ChatType, adderRole, newMemberRole MembershipRole, isOwner bool) bool {
	if chatType == ChatTypePrivate {
		return false
	}

	if isOwner {
		return IsRoleAllowedForChatType(chatType, newMemberRole)
	}

	if adderRole == RoleAdmin && chatType == ChatTypeServer {
		return newMemberRole == RoleMember || newMemberRole == RoleMuted || newMemberRole == RoleBanned
	}

	return false
}

func CanRemoveMember(chatType ChatType, removerRole, targetRole MembershipRole, isRemoverOwner bool, isTargetOwner bool) bool {
	if chatType == ChatTypePrivate {
		return false
	}

	if isTargetOwner {
		return false
	}

	if isRemoverOwner {
		return true
	}

	if removerRole == RoleAdmin && chatType == ChatTypeServer {
		return targetRole == RoleMember || targetRole == RoleMuted || targetRole == RoleBanned
	}

	return false
}

func CanChangeRole(chatType ChatType, changerRole, targetRole, newRole MembershipRole, isChangerOwner bool) bool {
	if chatType == ChatTypePrivate {
		return false
	}

	if !IsRoleAllowedForChatType(chatType, newRole) {
		return false
	}

	if targetRole == RoleOwner {
		return false
	}

	if isChangerOwner {
		return true
	}

	if changerRole == RoleAdmin && chatType == ChatTypeServer {
		return newRole == RoleMember || newRole == RoleMuted || newRole == RoleBanned
	}

	return false
}

func CanSendMessages(member *ChatMember) bool {
	if member == nil || !member.IsActive() {
		return false
	}
	if member.IsBanned() {
		return false
	}
	if member.IsMuted() {
		return false
	}
	return true
}

func CanViewChat(member *ChatMember) bool {
	if member == nil {
		return false
	}
	if member.IsBanned() {
		return false
	}
	return member.IsActive()
}

func CanOwnerLeave(chatType ChatType, memberCount int, isOwner bool) error {
	if !isOwner {
		return nil
	}

	if memberCount <= 1 {
		return ErrOwnerCannotLeave
	}

	if chatType == ChatTypePrivate {
		return ErrOwnerCannotLeave
	}

	return nil
}
