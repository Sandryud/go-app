-- 000005_create_password_resets_table.up.sql
-- Таблица для хранения кодов сброса пароля.

CREATE TABLE IF NOT EXISTS password_resets (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user_id
    ON password_resets (user_id);

CREATE INDEX IF NOT EXISTS idx_password_resets_expires_at
    ON password_resets (expires_at);

CREATE INDEX IF NOT EXISTS idx_password_resets_used_at
    ON password_resets (used_at);

COMMENT ON TABLE password_resets IS 'Коды сброса пароля для пользователей';
COMMENT ON COLUMN password_resets.user_id IS 'ID пользователя, которому принадлежит код';
COMMENT ON COLUMN password_resets.code_hash IS 'Хэш одноразового кода сброса пароля';
COMMENT ON COLUMN password_resets.expires_at IS 'Время, после которого код становится недействительным';
COMMENT ON COLUMN password_resets.attempts IS 'Количество использованных попыток ввода кода';
COMMENT ON COLUMN password_resets.max_attempts IS 'Максимально допустимое количество попыток';
COMMENT ON COLUMN password_resets.created_at IS 'Время создания записи с кодом';
COMMENT ON COLUMN password_resets.used_at IS 'Время использования кода (NULL, если код ещё не использован)';

