DROP TRIGGER IF EXISTS trigger_users_update_updated_at ON identity.users;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS contacts CASCADE;
DROP TABLE IF EXISTS users CASCADE;