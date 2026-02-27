package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	repo "workout-app/internal/repository/interfaces"
)

// exercisesMeta — минимальная структура для извлечения meta.version из JSON каталога.
type exercisesMeta struct {
	Meta struct {
		Version string `json:"version"`
	} `json:"meta"`
}

// ExercisesRepository реализует чтение каталога упражнений из файла.
type ExercisesRepository struct {
	filePath string
}

// Убедимся на этапе компиляции, что структура реализует интерфейс.
var _ repo.ExercisesCatalogRepository = (*ExercisesRepository)(nil)

// NewExercisesRepository создаёт репозиторий, читающий каталог из указанного файла.
func NewExercisesRepository(filePath string) *ExercisesRepository {
	return &ExercisesRepository{filePath: filePath}
}

// GetVersion возвращает версию каталога из meta.version.
func (r *ExercisesRepository) GetVersion(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", repo.ErrExercisesNotFound
		}
		return "", fmt.Errorf("read exercises catalog file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var meta exercisesMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", fmt.Errorf("parse exercises catalog meta: %w", err)
	}
	return meta.Meta.Version, nil
}

// GetData возвращает сырой JSON каталога и версию.
func (r *ExercisesRepository) GetData(ctx context.Context) (rawJSON []byte, version string, err error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", repo.ErrExercisesNotFound
		}
		return nil, "", fmt.Errorf("read exercises catalog file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	var meta exercisesMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, "", fmt.Errorf("parse exercises catalog meta: %w", err)
	}
	return data, meta.Meta.Version, nil
}
