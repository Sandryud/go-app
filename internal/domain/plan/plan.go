package plan

import (
	"time"

	"github.com/google/uuid"
)

// PlanCategory — категория плана тренировок (для каталога и фильтрации).
type PlanCategory string

const (
	PlanCategoryMassGain   PlanCategory = "mass_gain"   // набор мышечной массы
	PlanCategoryStrength   PlanCategory = "strength"    // сила
	PlanCategoryWeightLoss PlanCategory = "weight_loss" // сушка / похудение
	PlanCategoryEndurance  PlanCategory = "endurance"   // выносливость
	PlanCategoryGeneral    PlanCategory = "general"     // общая подготовка
)

// PlanLevel — уровень подготовки, для которого подходит план.
type PlanLevel string

const (
	PlanLevelBeginner     PlanLevel = "beginner"
	PlanLevelIntermediate PlanLevel = "intermediate"
	PlanLevelAdvanced     PlanLevel = "advanced"
)

// AllCategories — все допустимые категории плана (для валидации и API).
var AllCategories = []PlanCategory{
	PlanCategoryMassGain,
	PlanCategoryStrength,
	PlanCategoryWeightLoss,
	PlanCategoryEndurance,
	PlanCategoryGeneral,
}

// AllLevels — все допустимые уровни плана (для валидации и API).
var AllLevels = []PlanLevel{
	PlanLevelBeginner,
	PlanLevelIntermediate,
	PlanLevelAdvanced,
}

// Plan представляет доменную модель плана тренировок.
// Days заполняется только при запросе полного дерева (GetByID с вложенными днями и упражнениями).
type Plan struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Name          string
	IsActive      bool
	IsPublic      bool
	Category      *PlanCategory
	Level         *PlanLevel
	SourcePlanID  *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Days          []PlanDay // опционально: заполняется при GetByIDWithDaysAndExercises
}

// PlanDay — день плана (название, порядок).
// Exercises заполняется только при запросе полного дерева.
type PlanDay struct {
	ID         uuid.UUID
	PlanID     uuid.UUID
	Name       string
	SortOrder  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Exercises  []PlanDayExercise // опционально: заполняется при GetByIDWithDaysAndExercises
}

// PlanDayExercise — упражнение в дне плана (ссылка на каталог по exercise_id, параметры, порядок).
type PlanDayExercise struct {
	ID               uuid.UUID
	DayID            uuid.UUID
	ExerciseID       string
	Sets             int
	Reps             *int
	WeightKg         *float64
	DurationSeconds  *int
	DistanceMeters   *int
	RestSeconds      *int
	IsSuperset       bool
	SupersetGroup    *int
	SortOrder        int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
