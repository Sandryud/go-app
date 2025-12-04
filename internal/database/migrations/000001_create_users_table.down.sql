-- 000001_create_users_table.down.sql
-- Откат создания таблицы users

DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS users;

