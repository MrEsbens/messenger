package domain

type ChatType string

const (
	ChatTypePrivate ChatType = "private" // 1:1 чат
	ChatTypeGroup   ChatType = "group"   // Групповой чат (до ~100 участников)
	ChatTypeServer  ChatType = "server"  // Канал сервера (тысячи участников)
)

func IsValidChatType(t ChatType) bool {
	switch t {
	case ChatTypePrivate, ChatTypeGroup, ChatTypeServer:
		return true
	default:
		return false
	}
}

type MembershipRole string

const (
	RoleOwner  MembershipRole = "owner"  // Создатель чата
	RoleAdmin  MembershipRole = "admin"  // Администратор
	RoleMember MembershipRole = "member" // Обычный участник
	RoleMuted  MembershipRole = "muted"  // Заглушен (не может писать)
	RoleBanned MembershipRole = "banned" // Забанен (не имеет доступа)
)

func IsValidMembershipRole(r MembershipRole) bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleMuted, RoleBanned:
		return true
	default:
		return false
	}
}

type NotificationMode string

const (
	NotifyAll      NotificationMode = "all"      // Все сообщения
	NotifyMentions NotificationMode = "mentions" // Только упоминания
	NotifyNone     NotificationMode = "none"     // Без уведомлений
)

func IsValidNotificationMode(m NotificationMode) bool {
	switch m {
	case NotifyAll, NotifyMentions, NotifyNone:
		return true
	default:
		return false
	}
}

const (
	MaxChatNameLength        = 255
	MaxChatDescriptionLength = 4000
	MaxInviteTokenLength     = 64
	MaxGroupChatMembers      = 100
	MaxServerChatMembers     = 10000
	MinPrivateChatMembers    = 2
	MaxPrivateChatMembers    = 2
	MaxInviteMaxUses         = 1000
	MinInviteMaxUses         = 1
	DefaultInviteExpiryDays  = 7
	MaxInviteExpiryDays      = 365
)

func GetMaxMembersForChatType(chatType ChatType) int {
	switch chatType {
	case ChatTypePrivate:
		return MaxPrivateChatMembers
	case ChatTypeGroup:
		return MaxGroupChatMembers
	case ChatTypeServer:
		return MaxServerChatMembers
	default:
		return 0
	}
}

func GetAllowedRolesForChatType(chatType ChatType) []MembershipRole {
	switch chatType {
	case ChatTypePrivate:
		return []MembershipRole{RoleMember}
	case ChatTypeGroup:
		return []MembershipRole{RoleOwner, RoleMember}
	case ChatTypeServer:
		return []MembershipRole{RoleOwner, RoleAdmin, RoleMember, RoleMuted, RoleBanned}
	default:
		return []MembershipRole{}
	}
}

func IsRoleAllowedForChatType(chatType ChatType, role MembershipRole) bool {
	allowed := GetAllowedRolesForChatType(chatType)
	for _, r := range allowed {
		if r == role {
			return true
		}
	}
	return false
}
