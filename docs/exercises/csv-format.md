# Формат CSV для каталога упражнений

Данный документ описывает CSV-файлы в `data/csv/`, используемые парсером `cmd/csv2json` для генерации единого JSON-каталога. Целевая структура JSON описана в [exercise-json-plan.md](exercise-json-plan.md).

## Список файлов

| Файл | Назначение |
|------|------------|
| exercises.csv | Основная таблица упражнений (плоские поля + metadata) |
| exercise_equipment.csv | Справочник оборудования (id → slug) |
| exercise_equipment_links.csv | Связь упражнение–оборудование |
| movement_patterns.csv | Справочник паттернов движения |
| exercise_movement_pattern_links.csv | Связь упражнение–паттерн |
| exercise_purposes.csv | Справочник целей |
| exercise_purpose_links.csv | Связь упражнение–цель |
| exercise_tags.csv | Справочник тегов |
| exercise_tag_links.csv | Связь упражнение–тег |
| exercise_skills.csv | Справочник навыков |
| exercise_skill_links.csv | Связь упражнение–навык |
| measurement_units.csv | Справочник единиц измерения |
| exercise_measurement_unit_links.csv | Связь упражнение–единица |
| exercise_instructions.csv | Пошаговые инструкции |
| muscle_activations.csv | Мышцы и уровень активации |
| media_assets.csv | Медиа (фото/видео) |
| warnings.csv | Предупреждения по безопасности |
| contraindications.csv | Противопоказания |
| exercise_mistakes.csv | Типичные ошибки |
| programming_notes.csv | Заметки по программированию |
| breathing_tips.csv | Подсказки по дыханию |
| exercise_references.csv | Связи с другими упражнениями (регрессия/прогрессия/альтернатива) |
| strength_exercises.csv | Параметры силовых упражнений |
| cardio_exercises.csv | Параметры кардио |
| mobility_exercises.csv | Параметры мобильности |

## Маппинг полей и переименования

При сборке JSON некоторые имена колонок CSV преобразуются в имена полей JSON:

| JSON | CSV (файл) | Примечание |
|------|------------|------------|
| min_experience_months | minimum_experience (exercises.csv) | |
| popularity | popularity_score (exercises.csv) | |
| body_part | related_body_part (warnings.csv) | |
| alt | alt_text (media_assets.csv) | |
| duration_s | duration (media_assets.csv) | число, опционально |
| search_keywords | metadata.searchKeywords (exercises.csv) | JSON в ячейке |

Остальные поля передаются 1:1 по имени (в snake_case в JSON).

## Основная таблица: exercises.csv

Колонки: `id`, `type`, `name`, `description`, `difficulty`, `category`, `primary_muscle_group`, `tracking_type`, `location`, `minimum_experience`, `popularity_score`, `is_verified`, `certification_source`, `version`, `created_at`, `updated_at`, `metadata`.

- **metadata** — JSON-объект. Парсер использует только поле `searchKeywords` (массив строк). Значение `expert` в колонке `difficulty` автоматически приводится к `advanced` в JSON.

## Справочники и связи

Связи задаются через таблицы `*_links` с колонками `exercise_id` и `*_id`. В JSON подставляются слаги из соответствующего справочника (по `id`).

- equipment_links + exercise_equipment → `equipment[]` (поле справочника: `equipment`)
- movement_pattern_links + movement_patterns → `movement_patterns[]` (поле: `pattern`)
- purpose_links + exercise_purposes → `purposes[]` (поле: `purpose`)
- tag_links + exercise_tags → `tags[]` (поле: `tag`)
- skill_links + exercise_skills → `skills[]` (поле: `skill`)
- measurement_unit_links + measurement_units → `measurement_units[]` (поле: `unit`)

## Таблицы «один ко многим»

По `exercise_id` собираются массивы в JSON:

- **exercise_instructions** — сортировка по `step_number`, в JSON идут значения `instruction` → `instructions[]`.
- **muscle_activations** — `muscle_type`, `muscle_name`, `activation_level` → объекты `{ type, name, activation }` в `muscles[]`.
- **media_assets** — строки как объекты в `media[]` с переименованием `alt_text`→`alt`, `duration`→`duration_s`.
- **warnings** — `related_body_part`→`body_part`; колонка `recommendations` — JSON (см. ниже) → `recommendations[]`.
- **contraindications** — колонка `alternatives` — JSON (см. ниже) → `alternatives[]`.
- **exercise_mistakes** — колонка `mistake` → `mistakes[]`.
- **programming_notes** — колонка `note` → `programming_notes[]`.
- **breathing_tips** — колонка `tip` → `breathing.tips[]` (объединяется с полями из strength_exercises/cardio_exercises).
- **exercise_references** — строки, где `from_exercise_id` = текущее упражнение: в JSON в `references[]` с полями `exercise_id` = `to_exercise_id`, `relationship`, `name`, `effectiveness_rating`.

## JSON в ячейках

### metadata (exercises.csv)

Пример: `{"tags":[], "searchKeywords":["жим гантелей","dumbbell bench press"], "estimatedDuration":300}`  
Используется только `searchKeywords`.

### programming (strength_exercises, cardio_exercises, mobility_exercises)

Пример:

```json
{
  "sets": {"min": 3, "max": 5},
  "reps": {"min": 6, "max": 12},
  "rest": {"min": 120, "max": 180},
  "tempo": "3-1-1-0",
  "intensity": {"min": 70, "max": 85},
  "hold": {"min": 30, "max": 60}
}
```

`hold` опционален (для mobility). В JSON маппится в `sets_min/max`, `reps_min/max`, `rest_sec_min/max`, `tempo`, `intensity_pct_min/max`, `hold_sec_min/max`.

### recommendations (warnings.csv)

Формат: `{"recommendations": ["Текст 1", "Текст 2"]}`.

### alternatives (contraindications.csv)

Формат: `{"alternatives": ["slug-упражнения", "или текст"]}`.

## Enum и границы (валидация)

Парсер не меняет значения; валидатор проверяет следующее:

- **type**: strength | cardio | mobility | plyometric  
- **difficulty**: beginner | intermediate | advanced (в CSV допускается expert → в JSON сохраняется как advanced)  
- **location**: gym | home | both  
- **warnings.level**: critical | important | notice  
- **contraindications.severity**: absolute | relative  
- **references.relationship**: regression | progression | alternative  
- **muscles.type**: primary | secondary | stabilizer  
- **media.type**: image | video  
- **strength.force**: push | pull  
- **popularity**, **activation**, **effectiveness_rating**: 0–100  
- **min_experience_months** ≥ 0  
- **references.exercise_id** должны существовать в каталоге

## Запуск парсера

Из корня репозитория:

```bash
go run ./cmd/csv2json -csv data/csv -out dist/exercises.json
```

С отключённой валидацией (только сборка JSON):

```bash
go run ./cmd/csv2json -csv data/csv -out dist/exercises.json -validate=false
```

В CI/CD рекомендуется запускать с валидацией по умолчанию.
