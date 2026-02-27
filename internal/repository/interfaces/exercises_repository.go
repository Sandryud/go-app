package interfaces

import (
	"context"
	"errors"
)

// ErrExercisesNotFound возвращается, когда файл каталога упражнений не найден.
var ErrExercisesNotFound = errors.New("exercises catalog not found")

// ExercisesCatalogRepository определяет контракт для чтения каталога упражнений
// (версия и сырой JSON). Реализация может быть файловой или иной.
type ExercisesCatalogRepository interface {
	// GetVersion возвращает версию каталога из meta.version.
	// Возвращает ErrExercisesNotFound, если файл отсутствует.
	GetVersion(ctx context.Context) (string, error)

	// GetData возвращает сырой JSON каталога и версию (для тела ответа и ETag).
	// Возвращает ErrExercisesNotFound, если файл отсутствует.
	GetData(ctx context.Context) (rawJSON []byte, version string, err error)
}
