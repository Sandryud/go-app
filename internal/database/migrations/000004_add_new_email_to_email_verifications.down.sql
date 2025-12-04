-- 000004_add_new_email_to_email_verifications.down.sql
-- Откат добавления поля new_email в таблицу email_verifications

DROP INDEX IF EXISTS idx_email_verifications_user_id_new_email;

ALTER TABLE email_verifications
    DROP COLUMN IF EXISTS new_email;

