-- 000005_create_workout_plans_tables.up.sql
-- Таблицы модуля «Планы тренировок»: планы, дни плана, упражнения в дне.

-- workout_plans: план тренировок (владелец, название, активный/публичный флаги, категория, уровень).
CREATE TABLE IF NOT EXISTS workout_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    is_public BOOLEAN NOT NULL DEFAULT false,
    category VARCHAR(50) NULL
        CHECK (category IS NULL OR category IN ('mass_gain', 'strength', 'weight_loss', 'endurance', 'general')),
    level VARCHAR(20) NULL
        CHECK (level IS NULL OR level IN ('beginner', 'intermediate', 'advanced')),
    source_plan_id UUID NULL REFERENCES workout_plans(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workout_plans_user_active
    ON workout_plans (user_id) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_workout_plans_user_id ON workout_plans (user_id);
CREATE INDEX IF NOT EXISTS idx_workout_plans_is_public ON workout_plans (is_public);
CREATE INDEX IF NOT EXISTS idx_workout_plans_is_public_category ON workout_plans (is_public, category) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_workout_plans_is_public_level ON workout_plans (is_public, level) WHERE is_public = true;

COMMENT ON TABLE workout_plans IS 'Планы тренировок пользователей';
COMMENT ON COLUMN workout_plans.user_id IS 'Владелец плана';
COMMENT ON COLUMN workout_plans.is_active IS 'Только один активный план на пользователя';
COMMENT ON COLUMN workout_plans.is_public IS 'План отображается в общем каталоге; выставлять может только admin';
COMMENT ON COLUMN workout_plans.source_plan_id IS 'Ссылка на оригинал при копировании плана';

-- workout_plan_days: дни плана (название, порядок).
CREATE TABLE IF NOT EXISTS workout_plan_days (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES workout_plans(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workout_plan_days_plan_id ON workout_plan_days (plan_id);
CREATE INDEX IF NOT EXISTS idx_workout_plan_days_plan_sort ON workout_plan_days (plan_id, sort_order);

COMMENT ON TABLE workout_plan_days IS 'Дни плана тренировок (порядок по sort_order)';

-- workout_plan_day_exercises: упражнение в дне (ссылка на каталог, параметры, порядок, суперсет).
CREATE TABLE IF NOT EXISTS workout_plan_day_exercises (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    day_id UUID NOT NULL REFERENCES workout_plan_days(id) ON DELETE CASCADE,
    exercise_id VARCHAR(100) NOT NULL,
    sets INT NOT NULL CHECK (sets >= 1 AND sets <= 20),
    reps INT NULL,
    weight_kg DECIMAL(6,2) NULL,
    duration_seconds INT NULL,
    distance_meters INT NULL,
    rest_seconds INT NULL,
    is_superset BOOLEAN NOT NULL DEFAULT false,
    superset_group INT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workout_plan_day_exercises_day_id ON workout_plan_day_exercises (day_id);
CREATE INDEX IF NOT EXISTS idx_workout_plan_day_exercises_day_sort ON workout_plan_day_exercises (day_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_workout_plan_day_exercises_day_superset ON workout_plan_day_exercises (day_id, superset_group) WHERE superset_group IS NOT NULL;

COMMENT ON TABLE workout_plan_day_exercises IS 'Упражнения в дне плана (exercise_id — slug из каталога)';
