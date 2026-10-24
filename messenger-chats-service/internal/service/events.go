package service

import "time"

type ChatCreatedEvent struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatUpdatedEvent struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatDeletedEvent struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	DeleterID string    `json:"deleter_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type MemberAddedEvent struct {
	Type    string `json:"type"`
	ChatID  string `json:"chat_id"`
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	AddedBy string `json:"added_by"`
}

type MemberRemovedEvent struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	RemovedBy string `json:"removed_by"`
}

type MemberLeftEvent struct {
	Type   string `json:"type"`
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
}

type MemberRoleChangedEvent struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	OldRole   string `json:"old_role"`
	NewRole   string `json:"new_role"`
	UpdatedBy string `json:"updated_by"`
}

type InviteAcceptedEvent struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	InvitedBy string `json:"invited_by"`
}
