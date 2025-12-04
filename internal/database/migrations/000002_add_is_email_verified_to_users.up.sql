-- 000002_add_is_email_verified_to_users.up.sql
-- Добавляет признак подтверждения email у пользователя.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_email_verified BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN users.is_email_verified IS 'Признак того, что email пользователя подтверждён';

CREATE INDEX IF NOT EXISTS idx_users_is_email_verified
    ON users (is_email_verified);

