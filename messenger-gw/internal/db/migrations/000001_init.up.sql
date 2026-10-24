-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- -- Пользователи
-- CREATE TABLE users (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     email TEXT UNIQUE NOT NULL,
--     password_hash TEXT NOT NULL,
--     name TEXT NOT NULL,
--     avatar_url TEXT,
--     status TEXT,
--     created_at TIMESTAMP DEFAULT now()
-- );

-- -- Контакты
-- CREATE TABLE contacts (
--     user_id UUID REFERENCES users(id) ON DELETE CASCADE,
--     contact_id UUID REFERENCES users(id) ON DELETE CASCADE,
--     created_at TIMESTAMP DEFAULT now(),
--     PRIMARY KEY (user_id, contact_id)
-- );

-- Личные чаты
CREATE TABLE private_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id UUID REFERENCES users(id) ON DELETE CASCADE,
    user2_id UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now(),
    UNIQUE (user1_id, user2_id)
);

-- Групповые чаты
CREATE TABLE group_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    owner_id UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE group_chat_members (
    chat_id UUID REFERENCES group_chats(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP DEFAULT now(),
    PRIMARY KEY (chat_id, user_id)
);

-- Серверы
CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    owner_id UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE server_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE server_chat_members (
    chat_id UUID REFERENCES server_chats(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP DEFAULT now(),
    PRIMARY KEY (chat_id, user_id)
);

-- Сообщения
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_type TEXT CHECK (chat_type IN ('private','group','server')),
    chat_id UUID NOT NULL,
    sender_id UUID REFERENCES users(id) ON DELETE CASCADE,
    content TEXT,
    attachment_url TEXT,
    created_at TIMESTAMP DEFAULT now()
);

-- Статусы сообщений
CREATE TABLE message_statuses (
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status TEXT CHECK (status IN ('sent','delivered','read')),
    updated_at TIMESTAMP DEFAULT now(),
    PRIMARY KEY (message_id, user_id)
);
