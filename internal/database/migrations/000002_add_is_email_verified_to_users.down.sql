-- 000002_add_is_email_verified_to_users.down.sql
-- Откат добавления признака подтверждения email

DROP INDEX IF EXISTS idx_users_is_email_verified;

ALTER TABLE users
    DROP COLUMN IF EXISTS is_email_verified;

