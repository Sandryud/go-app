package plans

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	plandomain "workout-app/internal/domain/plan"
	userdomain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
)

// Доменные ошибки usecase планов.
var (
	ErrPlanNotFound            = errors.New("plan not found")
	ErrForbidden               = errors.New("access denied to plan")
	ErrOnlyAdminCanPublishPlan = errors.New("only admin can set plan as public")
	ErrInvalidCategory         = errors.New("invalid plan category")
	ErrInvalidLevel            = errors.New("invalid plan level")
	ErrDayNotFound             = errors.New("day not found")
	ErrExerciseEntryNotFound   = errors.New("exercise entry not found")
	ErrInvalidExerciseID       = errors.New("exercise_id not found in catalog")
	// ErrInvalidSetsRange возвращается, когда количество подходов вне диапазона 1–20.
	ErrInvalidSetsRange = errors.New("sets must be between 1 and 20")
)

// Service — usecase для работы с планами тренировок.
type Service interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*plandomain.Plan, error)
	GetByID(ctx context.Context, planID, userID uuid.UUID) (*plandomain.Plan, error)
	Create(ctx context.Context, userID uuid.UUID, callerRole userdomain.Role, input CreatePlanInput) (*plandomain.Plan, error)
	Update(ctx context.Context, planID, userID uuid.UUID, callerRole userdomain.Role, input UpdatePlanInput) (*plandomain.Plan, error)
	Delete(ctx context.Context, planID, userID uuid.UUID) error

	AddDay(ctx context.Context, planID, userID uuid.UUID, input CreateDayInput) (*plandomain.PlanDay, error)
	UpdateDay(ctx context.Context, planID, dayID, userID uuid.UUID, input UpdateDayInput) (*plandomain.PlanDay, error)
	DeleteDay(ctx context.Context, planID, dayID, userID uuid.UUID) error

	AddExerciseToDay(ctx context.Context, planID, dayID, userID uuid.UUID, input CreateDayExerciseInput) (*plandomain.PlanDayExercise, error)
	UpdateExerciseInDay(ctx context.Context, planID, dayID, exerciseEntryID, userID uuid.UUID, input UpdateDayExerciseInput) (*plandomain.PlanDayExercise, error)
	DeleteExerciseFromDay(ctx context.Context, planID, dayID, exerciseEntryID, userID uuid.UUID) error
}

// CreatePlanInput — вход создания плана.
type CreatePlanInput struct {
	Name      string
	IsActive  *bool
	IsPublic  *bool
	Category  *string
	Level     *string
}

// UpdatePlanInput — вход обновления плана (все поля опциональны).
type UpdatePlanInput struct {
	Name     *string
	IsActive *bool
	IsPublic *bool
	Category *string
	Level    *string
}

// CreateDayInput — вход добавления дня в план.
type CreateDayInput struct {
	Name      string
	SortOrder *int
}

// UpdateDayInput — вход обновления дня (все поля опциональны).
type UpdateDayInput struct {
	Name      *string
	SortOrder *int
}

// CreateDayExerciseInput — вход добавления упражнения в день.
type CreateDayExerciseInput struct {
	ExerciseID      string
	Sets            int
	Reps            *int
	WeightKg        *float64
	DurationSeconds *int
	DistanceMeters  *int
	RestSeconds     *int
	IsSuperset      *bool
	SortOrder       *int
}

// UpdateDayExerciseInput — вход обновления упражнения в дне (все поля опциональны).
type UpdateDayExerciseInput struct {
	Sets            *int
	Reps            *int
	WeightKg        *float64
	DurationSeconds *int
	DistanceMeters  *int
	RestSeconds     *int
	IsSuperset      *bool
	SortOrder       *int
}

type service struct {
	repo   repo.PlanRepository
	catalog repo.ExercisesCatalogRepository
}

// NewService создаёт usecase планов.
func NewService(planRepo repo.PlanRepository, catalog repo.ExercisesCatalogRepository) Service {
	return &service{repo: planRepo, catalog: catalog}
}

// ListByUser возвращает список планов пользователя без дней.
func (s *service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*plandomain.Plan, error) {
	list, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list plans by user: %w", err)
	}
	return list, nil
}

// GetByID возвращает план с деревом дней и упражнений. Доступ: владелец или план публичный.
func (s *service) GetByID(ctx context.Context, planID, userID uuid.UUID) (*plandomain.Plan, error) {
	plan, err := s.repo.GetByIDWithDaysAndExercises(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID && !plan.IsPublic {
		return nil, ErrForbidden
	}
	return plan, nil
}

// Create создаёт план. При is_public: true разрешено только для admin.
func (s *service) Create(ctx context.Context, userID uuid.UUID, callerRole userdomain.Role, input CreatePlanInput) (*plandomain.Plan, error) {
	if input.IsPublic != nil && *input.IsPublic && callerRole != userdomain.RoleAdmin {
		return nil, ErrOnlyAdminCanPublishPlan
	}
	var category *plandomain.PlanCategory
	if input.Category != nil {
		if *input.Category == "" {
			category = nil
		} else {
			c := plandomain.PlanCategory(*input.Category)
			if !isValidCategory(c) {
				return nil, ErrInvalidCategory
			}
			category = &c
		}
	}
	var level *plandomain.PlanLevel
	if input.Level != nil {
		if *input.Level == "" {
			level = nil
		} else {
			l := plandomain.PlanLevel(*input.Level)
			if !isValidLevel(l) {
				return nil, ErrInvalidLevel
			}
			level = &l
		}
	}
	isActive := false
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	isPublic := false
	if input.IsPublic != nil {
		isPublic = *input.IsPublic
	}
	now := time.Now().UTC()
	plan := &plandomain.Plan{
		UserID:   userID,
		Name:     input.Name,
		IsActive: isActive,
		IsPublic: isPublic,
		Category: category,
		Level:    level,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, fmt.Errorf("create plan: %w", err)
	}
	return plan, nil
}

// Update обновляет план. Только владелец. При is_public: true — только admin. При is_active: true снимает флаг с остальных в одной транзакции.
func (s *service) Update(ctx context.Context, planID, userID uuid.UUID, callerRole userdomain.Role, input UpdatePlanInput) (*plandomain.Plan, error) {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	if input.IsPublic != nil && *input.IsPublic && callerRole != userdomain.RoleAdmin {
		return nil, ErrOnlyAdminCanPublishPlan
	}
	if input.Name != nil {
		plan.Name = *input.Name
	}
	if input.IsActive != nil {
		plan.IsActive = *input.IsActive
	}
	if input.IsPublic != nil {
		plan.IsPublic = *input.IsPublic
	}
	if input.Category != nil {
		if *input.Category == "" {
			plan.Category = nil
		} else {
			c := plandomain.PlanCategory(*input.Category)
			if !isValidCategory(c) {
				return nil, ErrInvalidCategory
			}
			plan.Category = &c
		}
	}
	if input.Level != nil {
		if *input.Level == "" {
			plan.Level = nil
		} else {
			l := plandomain.PlanLevel(*input.Level)
			if !isValidLevel(l) {
				return nil, ErrInvalidLevel
			}
			plan.Level = &l
		}
	}
	plan.UpdatedAt = time.Now().UTC()

	if input.IsActive != nil && *input.IsActive {
		if err := s.repo.UpdatePlanAndDeactivateOthers(ctx, plan); err != nil {
			return nil, fmt.Errorf("update plan and deactivate others: %w", err)
		}
	} else {
		if err := s.repo.Update(ctx, plan); err != nil {
			return nil, fmt.Errorf("update plan: %w", err)
		}
	}
	updated, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("get plan after update: %w", err)
	}
	return updated, nil
}

// Delete удаляет план. Только владелец.
func (s *service) Delete(ctx context.Context, planID, userID uuid.UUID) error {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrPlanNotFound
		}
		return fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return ErrForbidden
	}
	if err := s.repo.Delete(ctx, planID); err != nil {
		return fmt.Errorf("delete plan: %w", err)
	}
	return nil
}

// AddDay добавляет день в план. Только владелец плана.
func (s *service) AddDay(ctx context.Context, planID, userID uuid.UUID, input CreateDayInput) (*plandomain.PlanDay, error) {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	day := &plandomain.PlanDay{
		PlanID:    planID,
		Name:      input.Name,
		SortOrder: sortOrder,
	}
	if err := s.repo.CreateDay(ctx, planID, day); err != nil {
		return nil, fmt.Errorf("create day: %w", err)
	}
	return day, nil
}

// UpdateDay обновляет день. Только владелец плана.
func (s *service) UpdateDay(ctx context.Context, planID, dayID, userID uuid.UUID, input UpdateDayInput) (*plandomain.PlanDay, error) {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	day, err := s.repo.GetDayByID(ctx, dayID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrDayNotFound
		}
		return nil, fmt.Errorf("get day by id: %w", err)
	}
	if day.PlanID != planID {
		return nil, ErrDayNotFound
	}
	if input.Name != nil {
		day.Name = *input.Name
	}
	if input.SortOrder != nil {
		day.SortOrder = *input.SortOrder
	}
	day.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateDay(ctx, day); err != nil {
		return nil, fmt.Errorf("update day: %w", err)
	}
	updated, err := s.repo.GetDayByID(ctx, dayID)
	if err != nil {
		return nil, fmt.Errorf("get day after update: %w", err)
	}
	return updated, nil
}

// DeleteDay удаляет день. Только владелец плана.
func (s *service) DeleteDay(ctx context.Context, planID, dayID, userID uuid.UUID) error {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrPlanNotFound
		}
		return fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return ErrForbidden
	}
	day, err := s.repo.GetDayByID(ctx, dayID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrDayNotFound
		}
		return fmt.Errorf("get day by id: %w", err)
	}
	if day.PlanID != planID {
		return ErrDayNotFound
	}
	if err := s.repo.DeleteDay(ctx, dayID); err != nil {
		return fmt.Errorf("delete day: %w", err)
	}
	return nil
}

// AddExerciseToDay добавляет упражнение в день. Валидация: exercise_id в каталоге, tracking_type.
func (s *service) AddExerciseToDay(ctx context.Context, planID, dayID, userID uuid.UUID, input CreateDayExerciseInput) (*plandomain.PlanDayExercise, error) {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	day, err := s.repo.GetDayByID(ctx, dayID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrDayNotFound
		}
		return nil, fmt.Errorf("get day by id: %w", err)
	}
	if day.PlanID != planID {
		return nil, ErrDayNotFound
	}
	info, err := s.catalog.GetExerciseByID(ctx, input.ExerciseID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrInvalidExerciseID
		}
		return nil, fmt.Errorf("get exercise by id: %w", err)
	}
	ex := &plandomain.PlanDayExercise{
		ExerciseID: input.ExerciseID,
		Sets:       input.Sets,
		Reps:       input.Reps,
		WeightKg:   input.WeightKg,
		DurationSeconds: input.DurationSeconds,
		DistanceMeters:  input.DistanceMeters,
		RestSeconds:     input.RestSeconds,
		IsSuperset:      input.IsSuperset != nil && *input.IsSuperset,
		SortOrder:       0,
	}
	if input.SortOrder != nil {
		ex.SortOrder = *input.SortOrder
	}
	normalizeExerciseByTrackingType(info.TrackingType, ex)
	if ex.Sets < 1 || ex.Sets > 20 {
		return nil, ErrInvalidSetsRange
	}
	if err := s.repo.CreateDayExercise(ctx, dayID, ex); err != nil {
		return nil, fmt.Errorf("create day exercise: %w", err)
	}
	return ex, nil
}

// UpdateExerciseInDay обновляет параметры упражнения в дне.
func (s *service) UpdateExerciseInDay(ctx context.Context, planID, dayID, exerciseEntryID, userID uuid.UUID, input UpdateDayExerciseInput) (*plandomain.PlanDayExercise, error) {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	day, err := s.repo.GetDayByID(ctx, dayID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrDayNotFound
		}
		return nil, fmt.Errorf("get day by id: %w", err)
	}
	if day.PlanID != planID {
		return nil, ErrDayNotFound
	}
	ex, err := s.repo.GetDayExerciseByID(ctx, exerciseEntryID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrExerciseEntryNotFound
		}
		return nil, fmt.Errorf("get day exercise by id: %w", err)
	}
	if ex.DayID != dayID {
		return nil, ErrExerciseEntryNotFound
	}
	if input.Sets != nil {
		ex.Sets = *input.Sets
	}
	if input.Reps != nil {
		ex.Reps = input.Reps
	}
	if input.WeightKg != nil {
		ex.WeightKg = input.WeightKg
	}
	if input.DurationSeconds != nil {
		ex.DurationSeconds = input.DurationSeconds
	}
	if input.DistanceMeters != nil {
		ex.DistanceMeters = input.DistanceMeters
	}
	if input.RestSeconds != nil {
		ex.RestSeconds = input.RestSeconds
	}
	if input.IsSuperset != nil {
		ex.IsSuperset = *input.IsSuperset
	}
	if input.SortOrder != nil {
		ex.SortOrder = *input.SortOrder
	}
	if ex.Sets < 1 || ex.Sets > 20 {
		return nil, ErrInvalidSetsRange
	}
	ex.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateDayExercise(ctx, ex); err != nil {
		return nil, fmt.Errorf("update day exercise: %w", err)
	}
	updated, err := s.repo.GetDayExerciseByID(ctx, exerciseEntryID)
	if err != nil {
		return nil, fmt.Errorf("get day exercise after update: %w", err)
	}
	return updated, nil
}

// DeleteExerciseFromDay удаляет упражнение из дня.
func (s *service) DeleteExerciseFromDay(ctx context.Context, planID, dayID, exerciseEntryID, userID uuid.UUID) error {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrPlanNotFound
		}
		return fmt.Errorf("get plan by id: %w", err)
	}
	if plan.UserID != userID {
		return ErrForbidden
	}
	day, err := s.repo.GetDayByID(ctx, dayID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrDayNotFound
		}
		return fmt.Errorf("get day by id: %w", err)
	}
	if day.PlanID != planID {
		return ErrDayNotFound
	}
	ex, err := s.repo.GetDayExerciseByID(ctx, exerciseEntryID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrExerciseEntryNotFound
		}
		return fmt.Errorf("get day exercise by id: %w", err)
	}
	if ex.DayID != dayID {
		return ErrExerciseEntryNotFound
	}
	if err := s.repo.DeleteDayExercise(ctx, exerciseEntryID); err != nil {
		return fmt.Errorf("delete day exercise: %w", err)
	}
	return nil
}

// normalizeExerciseByTrackingType приводит поля в соответствие с tracking_type каталога (bodyweight -> weight_kg nil).
func normalizeExerciseByTrackingType(trackingType string, ex *plandomain.PlanDayExercise) {
	switch trackingType {
	case "bodyweight-reps", "bodyweight":
		ex.WeightKg = nil
	case "time":
		ex.Reps = nil
		ex.DistanceMeters = nil
	case "distance":
		ex.Reps = nil
	}
}

func isValidCategory(c plandomain.PlanCategory) bool {
	for _, v := range plandomain.AllCategories {
		if v == c {
			return true
		}
	}
	return false
}

func isValidLevel(l plandomain.PlanLevel) bool {
	for _, v := range plandomain.AllLevels {
		if v == l {
			return true
		}
	}
	return false
}
