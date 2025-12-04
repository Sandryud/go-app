-- 001_create_users_table.sql
-- Базовая таблица пользователей для фитнес-приложения.
--
-- Улучшения:
-- - Автоматическая генерация UUID
-- - CHECK constraints для enum-полей
-- - Автоматическое обновление updated_at
-- - Оптимизированные типы данных (VARCHAR с ограничениями)
-- - Частичные уникальные индексы для поддержки soft delete
-- - Дополнительные индексы для производительности
-- - Комментарии к столбцам

-- Создание расширения для генерации UUID (если требуется)
-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    username VARCHAR(50) NOT NULL,

    first_name VARCHAR(100),
    last_name VARCHAR(100),
    birth_date DATE CHECK (
        birth_date IS NULL OR 
        (birth_date <= CURRENT_DATE AND birth_date >= CURRENT_DATE - INTERVAL '150 years')
    ),
    gender TEXT,
    avatar_url TEXT,

    role TEXT NOT NULL DEFAULT 'user' 
        CHECK (role IN ('user', 'coach', 'admin')),
    training_level TEXT NOT NULL DEFAULT 'beginner' 
        CHECK (training_level IN ('beginner', 'intermediate', 'advanced')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Частичные уникальные индексы для поддержки soft delete
-- Позволяют повторно использовать email/username после мягкого удаления
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique 
    ON users (email) WHERE deleted_at IS NULL;
    
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_unique 
    ON users (username) WHERE deleted_at IS NULL;

-- Индексы для производительности
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role) WHERE deleted_at IS NULL;

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер для автоматического обновления updated_at
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'update_users_updated_at'
    ) THEN
        CREATE TRIGGER update_users_updated_at
            BEFORE UPDATE ON users
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END;
$$;

-- Комментарии к таблице и столбцам
COMMENT ON TABLE users IS 'Таблица пользователей фитнес-приложения';
COMMENT ON COLUMN users.id IS 'Уникальный идентификатор пользователя (UUID)';
COMMENT ON COLUMN users.email IS 'Email пользователя (используется для входа)';
COMMENT ON COLUMN users.password_hash IS 'Хэш пароля пользователя';
COMMENT ON COLUMN users.username IS 'Никнейм пользователя (уникальный)';
COMMENT ON COLUMN users.role IS 'Роль пользователя: user, coach, admin';
COMMENT ON COLUMN users.training_level IS 'Уровень подготовки: beginner, intermediate, advanced';
COMMENT ON COLUMN users.deleted_at IS 'Время мягкого удаления (NULL если пользователь активен)';

