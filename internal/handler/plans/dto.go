package plans

import (
	"time"
)

// CreatePlanRequest — тело запроса POST /api/v1/plans.
type CreatePlanRequest struct {
	Name     string  `json:"name" binding:"required,max=200"`
	IsActive *bool   `json:"is_active,omitempty"`
	IsPublic *bool   `json:"is_public,omitempty"`
	Category *string `json:"category,omitempty" binding:"omitempty,oneof=mass_gain strength weight_loss endurance general"`
	Level    *string `json:"level,omitempty" binding:"omitempty,oneof=beginner intermediate advanced"`
}

// UpdatePlanRequest — тело запроса PUT /api/v1/plans/:id (все поля опциональны). Пустая строка category/level сбрасывает значение.
type UpdatePlanRequest struct {
	Name     *string `json:"name,omitempty" binding:"omitempty,max=200"`
	IsActive *bool   `json:"is_active,omitempty"`
	IsPublic *bool   `json:"is_public,omitempty"`
	Category *string `json:"category,omitempty" binding:"omitempty,oneof=mass_gain strength weight_loss endurance general"`
	Level    *string `json:"level,omitempty" binding:"omitempty,oneof=beginner intermediate advanced"`
}

// PlanListItemResponse — элемент списка планов (без дней).
type PlanListItemResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	IsActive  bool    `json:"is_active"`
	Category  *string `json:"category,omitempty"`
	Level     *string `json:"level,omitempty"`
	CreatedAt string  `json:"created_at"`
}

// PlanDetailResponse — план с деревом дней и упражнений (GET /api/v1/plans/:id).
type PlanDetailResponse struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	IsActive  bool                     `json:"is_active"`
	Category  *string                  `json:"category,omitempty"`
	Level     *string                  `json:"level,omitempty"`
	Days      []PlanDayDetailResponse  `json:"days"`
}

// PlanDayDetailResponse — день в детальном ответе плана.
type PlanDayDetailResponse struct {
	ID        string                        `json:"id"`
	Name      string                        `json:"name"`
	SortOrder int                           `json:"sort_order"`
	Exercises []PlanDayExerciseItemResponse `json:"exercises"`
}

// PlanDayExerciseItemResponse — упражнение в дне (в детальном ответе).
type PlanDayExerciseItemResponse struct {
	ID              string  `json:"id"`
	ExerciseID      string  `json:"exercise_id"`
	Sets            int     `json:"sets"`
	Reps            *int    `json:"reps,omitempty"`
	WeightKg        *float64 `json:"weight_kg,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
	DistanceMeters  *int    `json:"distance_meters,omitempty"`
	RestSeconds     *int    `json:"rest_seconds,omitempty"`
	IsSuperset      bool    `json:"is_superset"`
	SupersetGroup   *int    `json:"superset_group,omitempty"`
	SortOrder       int     `json:"sort_order"`
}

// PlanCreatedResponse — ответ 201 при создании плана (без дней).
type PlanCreatedResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	IsActive  bool    `json:"is_active"`
	IsPublic  bool    `json:"is_public"`
	Category  *string `json:"category,omitempty"`
	Level     *string `json:"level,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// PlanUpdatedResponse — ответ 200 при обновлении плана (без дней).
type PlanUpdatedResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	IsActive  bool    `json:"is_active"`
	IsPublic  bool    `json:"is_public"`
	Category  *string `json:"category,omitempty"`
	Level     *string `json:"level,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// CreateDayRequest — тело запроса POST /api/v1/plans/:planId/days.
type CreateDayRequest struct {
	Name      string `json:"name" binding:"required,max=200"`
	SortOrder *int   `json:"sort_order,omitempty"`
}

// UpdateDayRequest — тело запроса PUT /api/v1/plans/:planId/days/:dayId.
type UpdateDayRequest struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,max=200"`
	SortOrder *int   `json:"sort_order,omitempty"`
}

// DayResponse — ответ с днём (после создания или обновления).
type DayResponse struct {
	ID        string `json:"id"`
	PlanID    string `json:"plan_id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateDayExerciseRequest — тело запроса POST /api/v1/plans/:planId/days/:dayId/exercises.
type CreateDayExerciseRequest struct {
	ExerciseID      string   `json:"exercise_id" binding:"required,max=100"`
	Sets            int      `json:"sets" binding:"required,min=1,max=20"`
	Reps            *int     `json:"reps,omitempty"`
	WeightKg        *float64 `json:"weight_kg,omitempty"`
	DurationSeconds *int     `json:"duration_seconds,omitempty"`
	DistanceMeters  *int     `json:"distance_meters,omitempty"`
	RestSeconds     *int     `json:"rest_seconds,omitempty"`
	IsSuperset      *bool    `json:"is_superset,omitempty"`
	SortOrder       *int     `json:"sort_order,omitempty"`
}

// UpdateDayExerciseRequest — тело запроса PUT .../exercises/:exerciseEntryId.
type UpdateDayExerciseRequest struct {
	Sets            *int     `json:"sets,omitempty" binding:"omitempty,min=1,max=20"`
	Reps            *int     `json:"reps,omitempty"`
	WeightKg        *float64 `json:"weight_kg,omitempty"`
	DurationSeconds *int     `json:"duration_seconds,omitempty"`
	DistanceMeters  *int     `json:"distance_meters,omitempty"`
	RestSeconds     *int     `json:"rest_seconds,omitempty"`
	IsSuperset      *bool    `json:"is_superset,omitempty"`
	SortOrder       *int     `json:"sort_order,omitempty"`
}

// DayExerciseResponse — ответ с записью упражнения в дне (после создания/обновления).
type DayExerciseResponse struct {
	ID              string   `json:"id"`
	DayID           string   `json:"day_id"`
	ExerciseID      string   `json:"exercise_id"`
	Sets            int      `json:"sets"`
	Reps            *int     `json:"reps,omitempty"`
	WeightKg        *float64 `json:"weight_kg,omitempty"`
	DurationSeconds *int     `json:"duration_seconds,omitempty"`
	DistanceMeters  *int     `json:"distance_meters,omitempty"`
	RestSeconds     *int     `json:"rest_seconds,omitempty"`
	IsSuperset      bool     `json:"is_superset"`
	SupersetGroup   *int     `json:"superset_group,omitempty"`
	SortOrder       int      `json:"sort_order"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
