-- 000005_create_workout_plans_tables.down.sql
-- Откат создания таблиц модуля «Планы тренировок» (порядок: дочерние таблицы первыми).

DROP TABLE IF EXISTS workout_plan_day_exercises;
DROP TABLE IF EXISTS workout_plan_days;
DROP TABLE IF EXISTS workout_plans;
