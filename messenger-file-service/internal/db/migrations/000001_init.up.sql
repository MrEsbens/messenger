SET search_path TO storage;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    owner_type VARCHAR(50) NOT NULL,
    bucket_name VARCHAR(100) NOT NULL,
    object_key VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    confirmed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,

    UNIQUE (bucket_name, object_key)
);

CREATE INDEX idx_files_owner ON files (owner_id, owner_type);
CREATE INDEX idx_files_lookup ON files (bucket_name, object_key);
CREATE INDEX idx_files_deleted ON files (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE INDEX idx_files_status_pending ON files(status) WHERE status = 'pending';

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_files_update_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


COMMENT ON TABLE files IS 'Metadata for files stored in MinIO. Shared across all services.';
COMMENT ON COLUMN files.owner_type IS 'Domain owner: user, chat, server, message';
COMMENT ON COLUMN files.bucket_name IS 'MinIO bucket: avatars, attachments, server-icons';