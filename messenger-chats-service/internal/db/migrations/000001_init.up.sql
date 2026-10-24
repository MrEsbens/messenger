CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE chat_type AS ENUM ('private', 'group', 'server');
CREATE TYPE membership_role AS ENUM ('owner', 'admin', 'member', 'muted', 'banned');

CREATE TABLE chat_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    allow_member_invite BOOLEAN DEFAULT true,
    allow_member_kick BOOLEAN DEFAULT false,
    allow_member_mute BOOLEAN DEFAULT true,
    
    auto_moderation_enabled BOOLEAN DEFAULT false,
    max_message_length INT DEFAULT 4000,
    
    default_notification_mode VARCHAR(20) DEFAULT 'all',
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    type chat_type NOT NULL,
    
    name VARCHAR(255),
    description TEXT,
    
    config_id UUID,
    
    created_by UUID NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    
    CONSTRAINT chk_chat_name CHECK (
        (type != 'private' AND name IS NOT NULL AND name != '') OR
        (type = 'private')
    )
);

CREATE INDEX idx_chats_type ON chats (type);
CREATE INDEX idx_chats_created_by ON chats (created_by);
CREATE INDEX idx_chats_deleted ON chats (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_chats_config ON chats (config_id) WHERE config_id IS NOT NULL;

CREATE TABLE chat_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    chat_id UUID NOT NULL,
    user_id UUID NOT NULL,
    
    role membership_role DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT NOW(),
    left_at TIMESTAMP,
    
    notification_mode VARCHAR(20) DEFAULT 'all',
    is_muted_until TIMESTAMP,
    
    last_read_message_id UUID,
    last_activity_at TIMESTAMP
);

CREATE UNIQUE INDEX idx_members_unique_active 
    ON chat_members (chat_id, user_id) 
    WHERE left_at IS NULL;

CREATE INDEX idx_members_chat ON chat_members (chat_id, role);
CREATE INDEX idx_members_user ON chat_members (user_id, left_at);
CREATE INDEX idx_members_active ON chat_members (chat_id) WHERE left_at IS NULL;
CREATE INDEX idx_members_muted ON chat_members (is_muted_until) WHERE is_muted_until IS NOT NULL;

CREATE TABLE chat_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL,
    created_by UUID NOT NULL,
    
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP,
    max_uses INT,
    uses_count INT DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT NOW(),
    used_at TIMESTAMP
);

CREATE INDEX idx_invites_token ON chat_invites (token);
CREATE INDEX idx_invites_chat ON chat_invites (chat_id, expires_at);

CREATE INDEX idx_invites_active ON chat_invites (expires_at, uses_count) 
    WHERE expires_at IS NULL;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_chat_configs_update_updated_at
    BEFORE UPDATE ON chat_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_chats_update_updated_at
    BEFORE UPDATE ON chats
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE chats IS 'Чаты: личные, групповые, серверные. Владеет Chats Service.';
COMMENT ON TABLE chat_members IS 'Участники чатов. Слабая связность с Identity Service.';
COMMENT ON COLUMN chat_members.left_at IS 'NULL = активный участник, не-NULL = покинул чат';
COMMENT ON TABLE chat_invites IS 'Приглашения по ссылке для входа в чат';