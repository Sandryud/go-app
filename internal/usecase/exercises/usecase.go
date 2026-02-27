package exercises

import (
	"context"

	repo "workout-app/internal/repository/interfaces"
)

// ErrCatalogNotFound возвращается, когда каталог упражнений не найден (файл отсутствует).
// Реэкспорт repo.ErrExercisesNotFound для использования в handler без зависимости от repository/interfaces.
var ErrCatalogNotFound = repo.ErrExercisesNotFound

// Service описывает usecase-слой для получения версии и данных каталога упражнений.
type Service interface {
	// GetVersion возвращает версию каталога (meta.version).
	GetVersion(ctx context.Context) (string, error)

	// GetData возвращает сырой JSON каталога и версию для ETag.
	// Ошибки репозитория (в т.ч. ErrExercisesNotFound) пробрасываются без изменения.
	GetData(ctx context.Context) (rawJSON []byte, version string, err error)
}

type service struct {
	repo repo.ExercisesCatalogRepository
}

// NewService создаёт сервис каталога упражнений.
func NewService(repo repo.ExercisesCatalogRepository) Service {
	return &service{repo: repo}
}

// GetVersion возвращает версию каталога.
func (s *service) GetVersion(ctx context.Context) (string, error) {
	return s.repo.GetVersion(ctx)
}

// GetData возвращает данные каталога и версию.
func (s *service) GetData(ctx context.Context) ([]byte, string, error) {
	return s.repo.GetData(ctx)
}
