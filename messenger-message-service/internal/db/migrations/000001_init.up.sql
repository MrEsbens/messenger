SET search_path TO messages;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL,
    sender_id UUID NOT NULL,
    content TEXT NOT NULL,
    content_type VARCHAR(50) NOT NULL DEFAULT 'text' 
        CHECK (content_type IN ('text', 'markdown', 'code')),
    reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE message_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL 
        CHECK (status IN ('sent', 'delivered', 'read')),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (message_id, user_id)
);

CREATE INDEX idx_messages_chat ON messages (chat_id, created_at);
CREATE INDEX idx_messages_sender ON messages (sender_id, created_at);
CREATE INDEX idx_messages_deleted ON messages (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_messages_reply ON messages (reply_to_id) WHERE reply_to_id IS NOT NULL;

CREATE INDEX idx_statuses_user ON message_statuses (user_id, updated_at);
CREATE INDEX idx_statuses_message ON message_statuses (message_id);
CREATE INDEX idx_statuses_lookup ON message_statuses (message_id, user_id, status);

-- Триггеры
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_update_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_statuses_update_updated_at
    BEFORE UPDATE ON message_statuses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Комментарии
COMMENT ON TABLE messages IS 'Сообщения чатов. Владеет Messages Service.';
COMMENT ON COLUMN messages.chat_id IS 'Ссылка на чат (Chats Service). Нет FK, слабая связность.';
COMMENT ON COLUMN messages.sender_id IS 'Отправитель (Identity Service). Нет FK, слабая связность.';
COMMENT ON COLUMN messages.content_type IS 'Тип контента: text, markdown, code';
COMMENT ON COLUMN messages.reply_to_id IS 'ID сообщения, на которое это ответ (опционально)';
COMMENT ON TABLE message_statuses IS 'Статусы прочтения сообщений (per-user).';
COMMENT ON COLUMN message_statuses.status IS 'Статус: sent, delivered, read';