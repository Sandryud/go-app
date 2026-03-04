package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domain "workout-app/internal/domain/plan"
	repo "workout-app/internal/repository/interfaces"
)

// pgPlan — ORM-модель таблицы workout_plans.
type pgPlan struct {
	ID           string     `gorm:"column:id;type:uuid;primaryKey"`
	UserID       string     `gorm:"column:user_id;type:uuid;not null"`
	Name         string     `gorm:"column:name;type:varchar(200);not null"`
	IsActive     bool       `gorm:"column:is_active;not null"`
	IsPublic     bool       `gorm:"column:is_public;not null"`
	Category     *string    `gorm:"column:category;type:varchar(50)"`
	Level        *string    `gorm:"column:level;type:varchar(20)"`
	SourcePlanID *string    `gorm:"column:source_plan_id;type:uuid"`
	CreatedAt    time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;type:timestamptz;not null"`
	Days         []pgPlanDay `gorm:"foreignKey:PlanID"`
}

func (pgPlan) TableName() string { return "workout_plans" }

// pgPlanDay — ORM-модель таблицы workout_plan_days.
type pgPlanDay struct {
	ID        string             `gorm:"column:id;type:uuid;primaryKey"`
	PlanID    string             `gorm:"column:plan_id;type:uuid;not null"`
	Name      string             `gorm:"column:name;type:varchar(200);not null"`
	SortOrder int                `gorm:"column:sort_order;not null"`
	CreatedAt time.Time          `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt time.Time          `gorm:"column:updated_at;type:timestamptz;not null"`
	Exercises []pgPlanDayExercise `gorm:"foreignKey:DayID"`
}

func (pgPlanDay) TableName() string { return "workout_plan_days" }

// pgPlanDayExercise — ORM-модель таблицы workout_plan_day_exercises.
type pgPlanDayExercise struct {
	ID              string     `gorm:"column:id;type:uuid;primaryKey"`
	DayID           string     `gorm:"column:day_id;type:uuid;not null"`
	ExerciseID      string     `gorm:"column:exercise_id;type:varchar(100);not null"`
	Sets            int        `gorm:"column:sets;not null"`
	Reps            *int       `gorm:"column:reps"`
	WeightKg         *float64   `gorm:"column:weight_kg;type:decimal(6,2)"`
	DurationSeconds  *int       `gorm:"column:duration_seconds"`
	DistanceMeters   *int       `gorm:"column:distance_meters"`
	RestSeconds      *int       `gorm:"column:rest_seconds"`
	IsSuperset      bool       `gorm:"column:is_superset;not null"`
	SupersetGroup   *int       `gorm:"column:superset_group"`
	SortOrder       int        `gorm:"column:sort_order;not null"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (pgPlanDayExercise) TableName() string { return "workout_plan_day_exercises" }

// PlanRepository реализует repo.PlanRepository на GORM/Postgres.
type PlanRepository struct {
	db *gorm.DB
}

var _ repo.PlanRepository = (*PlanRepository)(nil)

// NewPlanRepository создаёт репозиторий планов.
func NewPlanRepository(db *gorm.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

func pgPlanToDomain(m *pgPlan) (*domain.Plan, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, fmt.Errorf("parse plan id %q: %w", m.ID, err)
	}
	userID, err := uuid.Parse(m.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse plan user_id %q: %w", m.UserID, err)
	}
	var sourcePlanID *uuid.UUID
	if m.SourcePlanID != nil && *m.SourcePlanID != "" {
		sid, err := uuid.Parse(*m.SourcePlanID)
		if err != nil {
			return nil, fmt.Errorf("parse plan source_plan_id %q: %w", *m.SourcePlanID, err)
		}
		sourcePlanID = &sid
	}
	var category *domain.PlanCategory
	if m.Category != nil {
		c := domain.PlanCategory(*m.Category)
		category = &c
	}
	var level *domain.PlanLevel
	if m.Level != nil {
		l := domain.PlanLevel(*m.Level)
		level = &l
	}
	p := &domain.Plan{
		ID:           id,
		UserID:       userID,
		Name:         m.Name,
		IsActive:     m.IsActive,
		IsPublic:     m.IsPublic,
		Category:     category,
		Level:        level,
		SourcePlanID: sourcePlanID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if len(m.Days) > 0 {
		p.Days = make([]domain.PlanDay, 0, len(m.Days))
		for i := range m.Days {
			d, err := pgPlanDayToDomain(&m.Days[i])
			if err != nil {
				return nil, fmt.Errorf("plan day %d: %w", i, err)
			}
			p.Days = append(p.Days, *d)
		}
	}
	return p, nil
}

func pgPlanDayToDomain(m *pgPlanDay) (*domain.PlanDay, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, fmt.Errorf("parse day id %q: %w", m.ID, err)
	}
	planID, err := uuid.Parse(m.PlanID)
	if err != nil {
		return nil, fmt.Errorf("parse day plan_id %q: %w", m.PlanID, err)
	}
	d := &domain.PlanDay{
		ID:        id,
		PlanID:    planID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if len(m.Exercises) > 0 {
		d.Exercises = make([]domain.PlanDayExercise, 0, len(m.Exercises))
		for i := range m.Exercises {
			ex, err := pgPlanDayExerciseToDomain(&m.Exercises[i])
			if err != nil {
				return nil, fmt.Errorf("day exercise %d: %w", i, err)
			}
			d.Exercises = append(d.Exercises, *ex)
		}
	}
	return d, nil
}

func pgPlanDayExerciseToDomain(m *pgPlanDayExercise) (*domain.PlanDayExercise, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, fmt.Errorf("parse day exercise id %q: %w", m.ID, err)
	}
	dayID, err := uuid.Parse(m.DayID)
	if err != nil {
		return nil, fmt.Errorf("parse day exercise day_id %q: %w", m.DayID, err)
	}
	return &domain.PlanDayExercise{
		ID:              id,
		DayID:           dayID,
		ExerciseID:      m.ExerciseID,
		Sets:            m.Sets,
		Reps:            m.Reps,
		WeightKg:        m.WeightKg,
		DurationSeconds: m.DurationSeconds,
		DistanceMeters:  m.DistanceMeters,
		RestSeconds:     m.RestSeconds,
		IsSuperset:      m.IsSuperset,
		SupersetGroup:   m.SupersetGroup,
		SortOrder:       m.SortOrder,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}

func domainPlanToPg(p *domain.Plan) *pgPlan {
	m := &pgPlan{
		ID:        p.ID.String(),
		UserID:    p.UserID.String(),
		Name:      p.Name,
		IsActive:  p.IsActive,
		IsPublic:  p.IsPublic,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if p.Category != nil {
		s := string(*p.Category)
		m.Category = &s
	}
	if p.Level != nil {
		s := string(*p.Level)
		m.Level = &s
	}
	if p.SourcePlanID != nil {
		s := p.SourcePlanID.String()
		m.SourcePlanID = &s
	}
	return m
}

// Create создаёт план (без дней и упражнений). Если plan.ID не задан, генерируется новый UUID.
func (r *PlanRepository) Create(ctx context.Context, plan *domain.Plan) error {
	if plan.ID == uuid.Nil {
		plan.ID = uuid.New()
	}
	m := domainPlanToPg(plan)
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		return err
	}
	plan.CreatedAt = m.CreatedAt
	plan.UpdatedAt = m.UpdatedAt
	return nil
}

// GetByID возвращает план без дней.
func (r *PlanRepository) GetByID(ctx context.Context, planID uuid.UUID) (*domain.Plan, error) {
	var m pgPlan
	err := r.db.WithContext(ctx).Where("id = ?", planID.String()).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pgPlanToDomain(&m)
}

// GetByIDWithDaysAndExercises возвращает план с деревом дней и упражнений (сортировка по sort_order).
func (r *PlanRepository) GetByIDWithDaysAndExercises(ctx context.Context, planID uuid.UUID) (*domain.Plan, error) {
	var m pgPlan
	err := r.db.WithContext(ctx).
		Preload("Days", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("Days.Exercises", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Where("id = ?", planID.String()).
		Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pgPlanToDomain(&m)
}

// ListByUserID возвращает планы пользователя без дней.
func (r *PlanRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Plan, error) {
	var rows []pgPlan
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID.String()).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Plan, 0, len(rows))
	for i := range rows {
		p, err := pgPlanToDomain(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Update обновляет план (поля таблицы workout_plans).
func (r *PlanRepository) Update(ctx context.Context, plan *domain.Plan) error {
	m := domainPlanToPg(plan)
	result := r.db.WithContext(ctx).Model(&pgPlan{}).Where("id = ?", m.ID).Updates(map[string]interface{}{
		"name":           m.Name,
		"is_active":      m.IsActive,
		"is_public":      m.IsPublic,
		"category":       m.Category,
		"level":          m.Level,
		"source_plan_id": m.SourcePlanID,
		"updated_at":    time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// UpdatePlanAndDeactivateOthers в одной транзакции снимает is_active у остальных планов пользователя и обновляет данный план.
func (r *PlanRepository) UpdatePlanAndDeactivateOthers(ctx context.Context, plan *domain.Plan) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Снять флаг у всех планов пользователя, кроме текущего
		res := tx.Model(&pgPlan{}).
			Where("user_id = ? AND id != ?", plan.UserID.String(), plan.ID.String()).
			Update("is_active", false)
		if res.Error != nil {
			return res.Error
		}
		// Обновить текущий план
		m := domainPlanToPg(plan)
		res = tx.Model(&pgPlan{}).Where("id = ?", m.ID).Updates(map[string]interface{}{
			"name":           m.Name,
			"is_active":      m.IsActive,
			"is_public":      m.IsPublic,
			"category":       m.Category,
			"level":          m.Level,
			"source_plan_id": m.SourcePlanID,
			"updated_at":     time.Now().UTC(),
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return repo.ErrNotFound
		}
		return nil
	})
}

// Delete удаляет план (CASCADE удалит дни и упражнения).
func (r *PlanRepository) Delete(ctx context.Context, planID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", planID.String()).Delete(&pgPlan{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func domainPlanDayToPg(d *domain.PlanDay) *pgPlanDay {
	return &pgPlanDay{
		ID:        d.ID.String(),
		PlanID:    d.PlanID.String(),
		Name:      d.Name,
		SortOrder: d.SortOrder,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// CreateDay создаёт день в плане.
func (r *PlanRepository) CreateDay(ctx context.Context, planID uuid.UUID, day *domain.PlanDay) error {
	day.PlanID = planID
	if day.ID == uuid.Nil {
		day.ID = uuid.New()
	}
	now := time.Now().UTC()
	day.CreatedAt = now
	day.UpdatedAt = now
	m := domainPlanDayToPg(day)
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		return err
	}
	day.CreatedAt = m.CreatedAt
	day.UpdatedAt = m.UpdatedAt
	return nil
}

// GetDayByID возвращает день по id (без упражнений).
func (r *PlanRepository) GetDayByID(ctx context.Context, dayID uuid.UUID) (*domain.PlanDay, error) {
	var m pgPlanDay
	err := r.db.WithContext(ctx).Where("id = ?", dayID.String()).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pgPlanDayToDomain(&m)
}

// UpdateDay обновляет день.
func (r *PlanRepository) UpdateDay(ctx context.Context, day *domain.PlanDay) error {
	m := domainPlanDayToPg(day)
	result := r.db.WithContext(ctx).Model(&pgPlanDay{}).Where("id = ?", m.ID).Updates(map[string]interface{}{
		"name":       m.Name,
		"sort_order": m.SortOrder,
		"updated_at": time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// DeleteDay удаляет день (CASCADE удалит упражнения в дне).
func (r *PlanRepository) DeleteDay(ctx context.Context, dayID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", dayID.String()).Delete(&pgPlanDay{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func domainPlanDayExerciseToPg(ex *domain.PlanDayExercise) *pgPlanDayExercise {
	return &pgPlanDayExercise{
		ID:              ex.ID.String(),
		DayID:           ex.DayID.String(),
		ExerciseID:      ex.ExerciseID,
		Sets:            ex.Sets,
		Reps:            ex.Reps,
		WeightKg:        ex.WeightKg,
		DurationSeconds: ex.DurationSeconds,
		DistanceMeters:  ex.DistanceMeters,
		RestSeconds:     ex.RestSeconds,
		IsSuperset:      ex.IsSuperset,
		SupersetGroup:   ex.SupersetGroup,
		SortOrder:       ex.SortOrder,
		CreatedAt:       ex.CreatedAt,
		UpdatedAt:       ex.UpdatedAt,
	}
}

// CreateDayExercise добавляет упражнение в день.
func (r *PlanRepository) CreateDayExercise(ctx context.Context, dayID uuid.UUID, ex *domain.PlanDayExercise) error {
	ex.DayID = dayID
	if ex.ID == uuid.Nil {
		ex.ID = uuid.New()
	}
	now := time.Now().UTC()
	ex.CreatedAt = now
	ex.UpdatedAt = now
	m := domainPlanDayExerciseToPg(ex)
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		return err
	}
	ex.CreatedAt = m.CreatedAt
	ex.UpdatedAt = m.UpdatedAt
	return nil
}

// GetDayExerciseByID возвращает запись упражнения в дне по id.
func (r *PlanRepository) GetDayExerciseByID(ctx context.Context, exerciseEntryID uuid.UUID) (*domain.PlanDayExercise, error) {
	var m pgPlanDayExercise
	err := r.db.WithContext(ctx).Where("id = ?", exerciseEntryID.String()).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pgPlanDayExerciseToDomain(&m)
}

// UpdateDayExercise обновляет параметры упражнения в дне.
func (r *PlanRepository) UpdateDayExercise(ctx context.Context, ex *domain.PlanDayExercise) error {
	m := domainPlanDayExerciseToPg(ex)
	result := r.db.WithContext(ctx).Model(&pgPlanDayExercise{}).Where("id = ?", m.ID).Updates(map[string]interface{}{
		"sets":              m.Sets,
		"reps":              m.Reps,
		"weight_kg":         m.WeightKg,
		"duration_seconds":  m.DurationSeconds,
		"distance_meters":   m.DistanceMeters,
		"rest_seconds":      m.RestSeconds,
		"is_superset":       m.IsSuperset,
		"superset_group":    m.SupersetGroup,
		"sort_order":        m.SortOrder,
		"updated_at":        time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// DeleteDayExercise удаляет упражнение из дня.
func (r *PlanRepository) DeleteDayExercise(ctx context.Context, exerciseEntryID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", exerciseEntryID.String()).Delete(&pgPlanDayExercise{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}
