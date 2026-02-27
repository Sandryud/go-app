# План: JSON и Go-структуры для каталога упражнений

## Корневая структура JSON (без dictionaries и locale)

- **meta**: `version`, `generated_at`, `total_count` — метаданные набора. Поле `locale` не используется: переводы slug (equipment, tags и т.д.) выполняются на клиенте через enum/i18n.
- **exercises**: массив объектов упражнений.

Справочники (equipment, tags, purposes, movement_patterns, skills, measurement_units) в JSON не передаются — клиент резолвит slug в отображаемые названия по своим enum/переводам.

---

## Общие поля упражнения

| Поле | Тип | Описание |
|------|-----|----------|
| id | string | Уникальный slug |
| type | string | strength \| cardio \| mobility \| plyometric |
| name | string | Название |
| description | string | Описание |
| difficulty | string | beginner \| intermediate \| advanced |
| category | string | Подкатегория (напр. strength-compound) |
| primary_muscle_group | string | Группа мышц (chest, back, …) |
| tracking_type | string | weight-reps, bodyweight-reps, time, … |
| location | string | gym \| home \| both |
| min_experience_months | number | Минимальный опыт, месяцев |
| popularity | number | 0–100 |
| is_verified | bool | Сертификация (напр. NSCA) |
| equipment | string[] | Слаги оборудования |
| movement_patterns | string[] | Слаги паттернов |
| purposes | string[] | Слаги целей |
| tags | string[] | Слаги тегов |
| skills | string[] | Слаги навыков |
| measurement_units | string[] | кг, reps, seconds, … |
| instructions | string[] | Пошаговые инструкции |
| muscles | array | { type, name, activation } |
| media | array | { url, type, alt, duration_s, thumbnail_url, is_primary, sort_order } |
| warnings | array | { level, message, body_part, recommendations } |
| contraindications | array | { condition, severity, reason, alternatives } |
| mistakes | string[] | Типичные ошибки |
| programming_notes | string[] | Заметки по программированию |
| breathing | object | eccentric, concentric, description, tips |
| references | array | { exercise_id, relationship, name, effectiveness_rating } |
| search_keywords | string[] | Ключевые слова для поиска |
| strength / cardio / mobility | object | Типспецифичный блок (programming, force и т.д.) |

---

## Типспецифичные блоки (одинаковая логика для strength, cardio, mobility)

- **strength**: `force` (push/pull), `programming` (sets_min/max, reps_min/max, rest_sec_min/max, tempo, intensity_pct_min/max).
- **cardio** / **mobility**: те же по смыслу поля `programming` (при необходимости duration, hold_sec и т.д.).

---

## Примеры и код

- Образец одного упражнения: [data/examples/exercises.json](../data/examples/exercises.json).
- Go-структуры для этого образца: [data/examples/exercises-interfaces.go](../data/examples/exercises-interfaces.go).
