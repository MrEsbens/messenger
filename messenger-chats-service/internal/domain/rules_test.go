package domain_test

import (
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestValidateChatMemberLimit_Private(t *testing.T) {
	// Ровно 2 участника — ок
	err := domain.ValidateChatMemberLimit(domain.ChatTypePrivate, 1, 2)
	assert.NoError(t, err)

	// Больше 2 — ошибка
	err = domain.ValidateChatMemberLimit(domain.ChatTypePrivate, 2, 3)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPrivateChatLimit)
}

func TestValidateChatMemberLimit_Group(t *testing.T) {
	// До 100 — ок
	err := domain.ValidateChatMemberLimit(domain.ChatTypeGroup, 50, 51)
	assert.NoError(t, err)

	// Больше 100 — ошибка
	err = domain.ValidateChatMemberLimit(domain.ChatTypeGroup, 100, 101)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrGroupChatLimit)
}

func TestCanAddMember_Private(t *testing.T) {
	// В private нельзя добавлять
	can := domain.CanAddMember(domain.ChatTypePrivate, domain.RoleOwner, domain.RoleMember, true)
	assert.False(t, can)
}

func TestCanAddMember_Group(t *testing.T) {
	// Owner может добавлять
	can := domain.CanAddMember(domain.ChatTypeGroup, domain.RoleOwner, domain.RoleMember, true)
	assert.True(t, can)

	// Member не может добавлять
	can = domain.CanAddMember(domain.ChatTypeGroup, domain.RoleMember, domain.RoleMember, false)
	assert.False(t, can)
}

func TestCanAddMember_Server(t *testing.T) {
	// Owner может добавлять любого
	can := domain.CanAddMember(domain.ChatTypeServer, domain.RoleOwner, domain.RoleAdmin, true)
	assert.True(t, can)

	// Admin может добавлять только member/muted/banned
	can = domain.CanAddMember(domain.ChatTypeServer, domain.RoleAdmin, domain.RoleMember, false)
	assert.True(t, can)

	// Admin не может добавлять admin/owner
	can = domain.CanAddMember(domain.ChatTypeServer, domain.RoleAdmin, domain.RoleAdmin, false)
	assert.False(t, can)
}

func TestCanChangeRole_Private(t *testing.T) {
	// В private нельзя менять роли
	can := domain.CanChangeRole(domain.ChatTypePrivate, domain.RoleOwner, domain.RoleMember, domain.RoleAdmin, true)
	assert.False(t, can)
}

func TestCanChangeRole_Server(t *testing.T) {
	// Owner может менять все роли (кроме owner)
	can := domain.CanChangeRole(domain.ChatTypeServer, domain.RoleOwner, domain.RoleMember, domain.RoleAdmin, true)
	assert.True(t, can)

	// Нельзя изменить роль owner
	can = domain.CanChangeRole(domain.ChatTypeServer, domain.RoleOwner, domain.RoleOwner, domain.RoleAdmin, true)
	assert.False(t, can)

	// Admin может менять только member/muted/banned
	can = domain.CanChangeRole(domain.ChatTypeServer, domain.RoleAdmin, domain.RoleMember, domain.RoleMuted, false)
	assert.True(t, can)
}

func TestIsRoleAllowedForChatType(t *testing.T) {
	tests := []struct {
		chatType domain.ChatType
		role     domain.MembershipRole
		allowed  bool
	}{
		{domain.ChatTypePrivate, domain.RoleMember, true},
		{domain.ChatTypePrivate, domain.RoleAdmin, false},
		{domain.ChatTypeGroup, domain.RoleOwner, true},
		{domain.ChatTypeGroup, domain.RoleMember, true},
		{domain.ChatTypeGroup, domain.RoleAdmin, false},
		{domain.ChatTypeServer, domain.RoleOwner, true},
		{domain.ChatTypeServer, domain.RoleAdmin, true},
		{domain.ChatTypeServer, domain.RoleMember, true},
		{domain.ChatTypeServer, domain.RoleMuted, true},
		{domain.ChatTypeServer, domain.RoleBanned, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.chatType)+"_"+string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.allowed, domain.IsRoleAllowedForChatType(tt.chatType, tt.role))
		})
	}
}
