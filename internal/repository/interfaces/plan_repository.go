package interfaces

import (
	"context"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/plan"
)

// PlanRepository определяет контракт для работы с планами тренировок.
type PlanRepository interface {
	// Create создаёт план. Days и вложенные сущности не сохраняются этим методом.
	Create(ctx context.Context, plan *domain.Plan) error

	// GetByID возвращает план по id без дней и упражнений.
	// Возвращает (nil, ErrNotFound), если план не найден.
	GetByID(ctx context.Context, planID uuid.UUID) (*domain.Plan, error)

	// GetByIDWithDaysAndExercises возвращает план с вложенными днями и упражнениями в днях (полное дерево).
	// Дни и упражнения упорядочены по sort_order.
	// Возвращает (nil, ErrNotFound), если план не найден.
	GetByIDWithDaysAndExercises(ctx context.Context, planID uuid.UUID) (*domain.Plan, error)

	// ListByUserID возвращает все планы пользователя без дней (id, name, is_active, category, level, created_at, updated_at).
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Plan, error)

	// Update обновляет поля плана. Не затрагивает дни и упражнения.
	// Возвращает ErrNotFound, если план не найден.
	Update(ctx context.Context, plan *domain.Plan) error

	// UpdatePlanAndDeactivateOthers обновляет план и снимает флаг is_active у остальных планов пользователя (одна транзакция).
	// Используется при установке is_active = true.
	// Возвращает ErrNotFound, если план не найден.
	UpdatePlanAndDeactivateOthers(ctx context.Context, plan *domain.Plan) error

	// Delete удаляет план (CASCADE удалит дни и упражнения в днях).
	// Возвращает ErrNotFound, если план не найден.
	Delete(ctx context.Context, planID uuid.UUID) error

	// CreateDay создаёт день в плане.
	CreateDay(ctx context.Context, planID uuid.UUID, day *domain.PlanDay) error

	// GetDayByID возвращает день по id (с plan_id для проверки владельца).
	// Возвращает (nil, ErrNotFound), если день не найден.
	GetDayByID(ctx context.Context, dayID uuid.UUID) (*domain.PlanDay, error)

	// UpdateDay обновляет день (name, sort_order).
	// Возвращает ErrNotFound, если день не найден.
	UpdateDay(ctx context.Context, day *domain.PlanDay) error

	// DeleteDay удаляет день (CASCADE удалит упражнения в дне).
	// Возвращает ErrNotFound, если день не найден.
	DeleteDay(ctx context.Context, dayID uuid.UUID) error

	// CreateDayExercise добавляет упражнение в день.
	CreateDayExercise(ctx context.Context, dayID uuid.UUID, ex *domain.PlanDayExercise) error

	// GetDayExerciseByID возвращает запись упражнения в дне по id.
	// Возвращает (nil, ErrNotFound), если запись не найдена.
	GetDayExerciseByID(ctx context.Context, exerciseEntryID uuid.UUID) (*domain.PlanDayExercise, error)

	// UpdateDayExercise обновляет параметры упражнения в дне.
	// Возвращает ErrNotFound, если запись не найдена.
	UpdateDayExercise(ctx context.Context, ex *domain.PlanDayExercise) error

	// DeleteDayExercise удаляет упражнение из дня.
	// Возвращает ErrNotFound, если запись не найдена.
	DeleteDayExercise(ctx context.Context, exerciseEntryID uuid.UUID) error
}
