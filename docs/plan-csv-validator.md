# План: CSV-валидатор (только проверка CSV)

## Цель

Реализовать **только** CSV-валидатор, который проверяет все CSV-файлы с упражнениями в заданной директории. Путь к директории с CSV задаётся в скрипте (например, флагом CLI).

---

## Scope

- **Входит**: пакет валидации CSV + CLI, в котором путь к директории с CSV задаётся явно (флаг `-csv=...` или переменная в коде).
- **Не входит**: интеграция с csv2json, опциональная проверка каталога после парсинга (validator.Validate(catalog)).

---

## 1. Пакет `internal/exercises/csvvalidator`

- **Тип ошибки**: файл, строка (1-based), колонка, сообщение.
- **Вход**: путь к директории с CSV (строка).
- **Выход**: срез ошибок; при I/O-ошибках — возвращать ошибку + по возможности накопленные ошибки валидации.

Проверки (по приоритету):

1. **exercises.csv** (обязательный файл)
   - Наличие обязательных колонок: id, type, name, difficulty, primary_muscle_group, tracking_type, location.
   - По строкам: id не пустой и уникален; type ∈ {strength, cardio, mobility, plyometric}; difficulty ∈ {beginner, intermediate, advanced, expert}; location ∈ {gym, home, both}; minimum_experience ≥ 0; popularity_score ∈ [0, 100]; is_verified — bool; metadata — валидный JSON при непустом значении.
   - Собрать множество всех id для проверки ссылок в других файлах.

2. **Файлы один-ко-многим** (если файл есть)
   - exercise_instructions.csv, muscle_activations.csv, media_assets.csv, warnings.csv, contraindications.csv, exercise_mistakes.csv, programming_notes.csv, breathing_tips.csv: колонка exercise_id; каждое значение должно быть в множестве id из exercises.csv; проверка enum’ов и числовых диапазонов по колонкам (как в текущем парсере/валидаторе).
   - exercise_references.csv: from_exercise_id, to_exercise_id ∈ id из exercises.csv; relationship ∈ {regression, progression, alternative}; effectiveness_rating ∈ [0, 100].

3. **Справочники и link-таблицы** (если файлы есть)
   - Проверка обязательных колонок; в link-таблицах — exercise_id из exercises.csv, внешние ключи из соответствующих справочников.

4. **Тип-специфичные** (strength_exercises.csv, cardio_exercises.csv, mobility_exercises.csv)
   - exercise_id ∈ id из exercises.csv; type упражнения в exercises.csv совпадает с типом файла; при наличии — проверка JSON programming и enum force.

---

## 2. CLI `cmd/csv-validate`

- Флаг **`-csv`** (обязательный или с дефолтом, например `data/csv`): путь к директории с CSV.
- Чтение директории, вызов `csvvalidator.Validate(dir)`.
- Вывод каждой ошибки в stdout (формат: файл, строка, колонка, сообщение).
- При наличии хотя бы одной ошибки — exit code 1, иначе 0.

Пример вызова:

```bash
go run ./cmd/csv-validate -csv=./data/csv
# или свой путь
go run ./cmd/csv-validate -csv=/path/to/my/csv
```

---

## 3. Детали

- Переиспользовать enum-множества и правила из `internal/exercises/validator` (или вынести в общий пакет), чтобы не дублировать допустимые значения.
- Если какой-то из вспомогательных CSV отсутствует — не считать ошибкой (только exercises.csv обязателен).
- Кодировка: UTF-8, как в текущем парсере.

---

## 4. Тесты

- `internal/exercises/csvvalidator`: тесты на валидный exercises.csv; неверные enum’ы; дубликат id; несуществующий exercise_id в дочерних файлах; неверные диапазоны.
