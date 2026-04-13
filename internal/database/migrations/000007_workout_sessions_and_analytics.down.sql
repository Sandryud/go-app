-- 000007_workout_sessions_and_analytics.down.sql
-- Откат: дочерние таблицы первыми.

DROP TABLE IF EXISTS plan_day_workout_stats;
DROP TABLE IF EXISTS workout_session_reports;
DROP TABLE IF EXISTS workout_session_sets;
DROP TABLE IF EXISTS workout_session_exercise_slots;
DROP TABLE IF EXISTS workout_sessions;
