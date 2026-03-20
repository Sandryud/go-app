-- Публичные ссылки на план (share) и события копирования для метрик.

CREATE TABLE IF NOT EXISTS workout_plan_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES workout_plans(id) ON DELETE CASCADE,
    token UUID NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workout_plan_shares_plan_active
    ON workout_plan_shares (plan_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_workout_plan_shares_plan_id ON workout_plan_shares (plan_id);

COMMENT ON TABLE workout_plan_shares IS 'Публичная ссылка на план для шаринга (одна активная на план)';
COMMENT ON COLUMN workout_plan_shares.token IS 'Токен в URL /public/plans/by-share/:token';
COMMENT ON COLUMN workout_plan_shares.revoked_at IS 'Если задано — ссылка недействительна';

CREATE TABLE IF NOT EXISTS workout_plan_copy_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_plan_id UUID NULL REFERENCES workout_plans(id) ON DELETE SET NULL,
    copy_plan_id UUID NOT NULL REFERENCES workout_plans(id) ON DELETE CASCADE,
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('share')),
    share_id UUID NULL REFERENCES workout_plan_shares(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workout_plan_copy_events_source ON workout_plan_copy_events (source_plan_id);
CREATE INDEX IF NOT EXISTS idx_workout_plan_copy_events_source_recipient
    ON workout_plan_copy_events (source_plan_id, recipient_user_id);

COMMENT ON TABLE workout_plan_copy_events IS 'Событие «забрать план себе» для аналитики';
