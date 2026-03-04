package interfaces

import (
	"context"
	"errors"
)

// ErrExercisesNotFound возвращается, когда файл каталога упражнений не найден.
var ErrExercisesNotFound = errors.New("exercises catalog not found")

// ExerciseInfo — минимальная информация об упражнении из каталога (для валидации в планах).
type ExerciseInfo struct {
	ID           string // slug (exercise_id)
	TrackingType string // weight-reps, bodyweight-reps, time, distance и т.д.
}

// ExercisesCatalogRepository определяет контракт для чтения каталога упражнений
// (версия и сырой JSON). Реализация может быть файловой или иной.
type ExercisesCatalogRepository interface {
	// GetVersion возвращает версию каталога из meta.version.
	// Возвращает ErrExercisesNotFound, если файл отсутствует.
	GetVersion(ctx context.Context) (string, error)

	// GetData возвращает сырой JSON каталога и версию (для тела ответа и ETag).
	// Возвращает ErrExercisesNotFound, если файл отсутствует.
	GetData(ctx context.Context) (rawJSON []byte, version string, err error)

	// GetExerciseByID возвращает упражнение по id (slug) из каталога.
	// Возвращает (nil, ErrNotFound), если упражнение не найдено.
	GetExerciseByID(ctx context.Context, exerciseID string) (*ExerciseInfo, error)
}
