-- 000004_add_new_email_to_email_verifications.up.sql
-- Добавление поля new_email в таблицу email_verifications для поддержки изменения email.

ALTER TABLE email_verifications
    ADD COLUMN IF NOT EXISTS new_email VARCHAR(255) NULL;

CREATE INDEX IF NOT EXISTS idx_email_verifications_user_id_new_email
    ON email_verifications (user_id, new_email)
    WHERE new_email IS NOT NULL;

COMMENT ON COLUMN email_verifications.new_email IS 'Новый email для изменения (NULL при обычном подтверждении при регистрации)';

