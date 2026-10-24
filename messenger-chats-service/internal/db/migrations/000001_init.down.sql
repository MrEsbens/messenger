SET search_path TO chats;

DROP TRIGGER IF EXISTS trigger_chats_update_updated_at ON chats;
DROP TRIGGER IF EXISTS trigger_chat_configs_update_updated_at ON chat_configs;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS chat_invites;
DROP TABLE IF EXISTS chat_members;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS chat_configs;

DROP TYPE IF EXISTS membership_role;
DROP TYPE IF EXISTS chat_type;