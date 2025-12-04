-- 003_create_email_verifications_table.sql
-- Таблица для хранения кодов подтверждения email.

CREATE TABLE IF NOT EXISTS email_verifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_user_id
    ON email_verifications (user_id);

CREATE INDEX IF NOT EXISTS idx_email_verifications_expires_at
    ON email_verifications (expires_at);

COMMENT ON TABLE email_verifications IS 'Коды подтверждения email для пользователей';
COMMENT ON COLUMN email_verifications.user_id IS 'ID пользователя, которому принадлежит код';
COMMENT ON COLUMN email_verifications.code_hash IS 'Хэш одноразового кода подтверждения email';
COMMENT ON COLUMN email_verifications.expires_at IS 'Время, после которого код становится недействительным';
COMMENT ON COLUMN email_verifications.attempts IS 'Количество использованных попыток ввода кода';
COMMENT ON COLUMN email_verifications.max_attempts IS 'Максимально допустимое количество попыток';
COMMENT ON COLUMN email_verifications.created_at IS 'Время создания записи с кодом';


