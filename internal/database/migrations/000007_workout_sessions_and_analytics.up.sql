-- 000007_workout_sessions_and_analytics.up.sql
-- Тренировочные сессии: снимок дня плана, подходы, отчёт; агрегаты для аналитики по дням плана.

-- workout_sessions: одна сессия = один проход по дню плана (владелец, ссылки на план/день, статус, отчёт).
CREATE TABLE IF NOT EXISTS workout_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES workout_plans(id) ON DELETE CASCADE,
    plan_day_id UUID NOT NULL REFERENCES workout_plan_days(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress'
        CHECK (status IN ('in_progress', 'completed', 'cancelled')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ NULL,
    cancelled_at TIMESTAMPTZ NULL,
    report_status VARCHAR(20) NOT NULL DEFAULT 'not_applicable'
        CHECK (report_status IN ('not_applicable', 'pending', 'ready', 'failed')),
    report_computed_at TIMESTAMPTZ NULL,
    report_error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workout_sessions_user_id ON workout_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_workout_sessions_plan_id ON workout_sessions (plan_id);
CREATE INDEX IF NOT EXISTS idx_workout_sessions_plan_day_id ON workout_sessions (plan_day_id);
CREATE INDEX IF NOT EXISTS idx_workout_sessions_user_plan_status_ended
    ON workout_sessions (user_id, plan_id, status, ended_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_workout_sessions_user_status_ended
    ON workout_sessions (user_id, status, ended_at DESC NULLS LAST);

COMMENT ON TABLE workout_sessions IS 'Сессия выполнения тренировки по дню плана';
COMMENT ON COLUMN workout_sessions.report_status IS 'not_applicable до complete/cancel; pending — отчёт в очереди/считается';

-- workout_session_exercise_slots: снимок слота дня на старт сессии (плановые цели + текущее упражнение после замены).
CREATE TABLE IF NOT EXISTS workout_session_exercise_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES workout_sessions(id) ON DELETE CASCADE,
    plan_day_exercise_id UUID NOT NULL REFERENCES workout_plan_day_exercises(id) ON DELETE RESTRICT,
    sort_order INT NOT NULL DEFAULT 0,
    planned_sets INT NOT NULL CHECK (planned_sets >= 1 AND planned_sets <= 20),
    planned_reps INT NULL,
    planned_weight_kg DECIMAL(6,2) NULL,
    planned_duration_seconds INT NULL,
    planned_distance_meters INT NULL,
    planned_rest_seconds INT NULL,
    initial_exercise_id VARCHAR(100) NOT NULL,
    current_exercise_id VARCHAR(100) NOT NULL,
    replacement_count INT NOT NULL DEFAULT 0 CHECK (replacement_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, plan_day_exercise_id)
);

CREATE INDEX IF NOT EXISTS idx_workout_session_slots_session_id
    ON workout_session_exercise_slots (session_id);
CREATE INDEX IF NOT EXISTS idx_workout_session_slots_session_sort
    ON workout_session_exercise_slots (session_id, sort_order);

COMMENT ON TABLE workout_session_exercise_slots IS 'Слот упражнения в сессии (снимок плана); замена меняет current_exercise_id';
COMMENT ON COLUMN workout_session_exercise_slots.plan_day_exercise_id IS 'RESTRICT: нельзя удалить строку плана, если на неё ссылается сессия';

-- workout_session_sets: фактические подходы (performed_exercise_id учитывает замену).
CREATE TABLE IF NOT EXISTS workout_session_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES workout_sessions(id) ON DELETE CASCADE,
    slot_id UUID NOT NULL REFERENCES workout_session_exercise_slots(id) ON DELETE CASCADE,
    set_index INT NOT NULL CHECK (set_index >= 1 AND set_index <= 20),
    performed_exercise_id VARCHAR(100) NOT NULL,
    reps INT NULL,
    weight_kg DECIMAL(6,2) NULL,
    duration_seconds INT NULL,
    distance_meters INT NULL,
    rest_after_seconds INT NULL,
    logged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    client_request_id VARCHAR(100) NULL,
    UNIQUE (slot_id, set_index)
);

CREATE INDEX IF NOT EXISTS idx_workout_session_sets_session_slot_index
    ON workout_session_sets (session_id, slot_id, set_index);
CREATE INDEX IF NOT EXISTS idx_workout_session_sets_performed_session
    ON workout_session_sets (performed_exercise_id, session_id);

COMMENT ON TABLE workout_session_sets IS 'Залогированный подход; performed_exercise_id — slug после возможной замены';
COMMENT ON COLUMN workout_session_sets.client_request_id IS 'Опционально: идемпотентность записи подхода с клиента';

-- workout_session_reports: результат асинхронного расчёта отчёта (1:1 с завершённой/отменённой сессией при наличии отчёта).
CREATE TABLE IF NOT EXISTS workout_session_reports (
    session_id UUID PRIMARY KEY REFERENCES workout_sessions(id) ON DELETE CASCADE,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    completion_percent DECIMAL(5,2) NULL,
    total_sets_completed INT NULL,
    total_volume DECIMAL(14,2) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE workout_session_reports IS 'Итоговый отчёт тренировки (воркер Asynq)';

-- plan_day_workout_stats: денормализация для списка дней плана со статусом (последняя релевантная сессия).
CREATE TABLE IF NOT EXISTS plan_day_workout_stats (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES workout_plans(id) ON DELETE CASCADE,
    plan_day_id UUID NOT NULL REFERENCES workout_plan_days(id) ON DELETE CASCADE,
    last_session_id UUID NULL REFERENCES workout_sessions(id) ON DELETE SET NULL,
    last_completed_at TIMESTAMPTZ NULL,
    last_completion_percent DECIMAL(5,2) NULL,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, plan_id, plan_day_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_day_workout_stats_plan_id ON plan_day_workout_stats (plan_id);

COMMENT ON TABLE plan_day_workout_stats IS 'Агрегат аналитики: последняя сессия и процент по дню плана для пользователя';
