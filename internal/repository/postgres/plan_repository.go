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

// pgPlanShare — ORM-модель таблицы workout_plan_shares.
type pgPlanShare struct {
	ID        string     `gorm:"column:id;type:uuid;primaryKey"`
	PlanID    string     `gorm:"column:plan_id;type:uuid;not null"`
	Token     string     `gorm:"column:token;type:uuid;not null;uniqueIndex"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	RevokedAt *time.Time `gorm:"column:revoked_at;type:timestamptz"`
}

func (pgPlanShare) TableName() string { return "workout_plan_shares" }

// pgPlanCopyEvent — ORM-модель таблицы workout_plan_copy_events.
type pgPlanCopyEvent struct {
	ID              string    `gorm:"column:id;type:uuid;primaryKey"`
	SourcePlanID    *string   `gorm:"column:source_plan_id;type:uuid"`
	CopyPlanID      string    `gorm:"column:copy_plan_id;type:uuid;not null"`
	RecipientUserID string    `gorm:"column:recipient_user_id;type:uuid;not null"`
	Channel         string    `gorm:"column:channel;type:varchar(20);not null"`
	ShareID         *string   `gorm:"column:share_id;type:uuid"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;not null"`
}

func (pgPlanCopyEvent) TableName() string { return "workout_plan_copy_events" }

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
	return r.CreateDayExercises(ctx, dayID, []*domain.PlanDayExercise{ex})
}

// CreateDayExercises добавляет несколько упражнений в день в одной транзакции.
func (r *PlanRepository) CreateDayExercises(ctx context.Context, dayID uuid.UUID, exercises []*domain.PlanDayExercise) error {
	if len(exercises) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.createDayExercisesWithDB(tx, dayID, exercises)
	})
}

// createDayExercisesWithDB выполняет вставку записей упражнений, используя переданный *gorm.DB.
func (r *PlanRepository) createDayExercisesWithDB(db *gorm.DB, dayID uuid.UUID, exercises []*domain.PlanDayExercise) error {
	now := time.Now().UTC()

	pgItems := make([]pgPlanDayExercise, 0, len(exercises))
	for _, ex := range exercises {
		ex.DayID = dayID
		if ex.ID == uuid.Nil {
			ex.ID = uuid.New()
		}
		ex.CreatedAt = now
		ex.UpdatedAt = now
		m := domainPlanDayExerciseToPg(ex)
		pgItems = append(pgItems, *m)
	}

	if err := db.Create(&pgItems).Error; err != nil {
		return err
	}

	for i := range exercises {
		exercises[i].CreatedAt = pgItems[i].CreatedAt
		exercises[i].UpdatedAt = pgItems[i].UpdatedAt
	}

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

func pgShareToDomain(m *pgPlanShare) (*domain.PlanShare, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, fmt.Errorf("parse share id: %w", err)
	}
	planID, err := uuid.Parse(m.PlanID)
	if err != nil {
		return nil, fmt.Errorf("parse share plan_id: %w", err)
	}
	token, err := uuid.Parse(m.Token)
	if err != nil {
		return nil, fmt.Errorf("parse share token: %w", err)
	}
	return &domain.PlanShare{
		ID:        id,
		PlanID:    planID,
		Token:     token,
		CreatedAt: m.CreatedAt,
		RevokedAt: m.RevokedAt,
	}, nil
}

// GetActiveShareByPlanID возвращает активную share-запись для плана.
func (r *PlanRepository) GetActiveShareByPlanID(ctx context.Context, planID uuid.UUID) (*domain.PlanShare, error) {
	var m pgPlanShare
	err := r.db.WithContext(ctx).
		Where("plan_id = ? AND revoked_at IS NULL", planID.String()).
		Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pgShareToDomain(&m)
}

// CreateShare создаёт новую share-запись (вызвать только если активной ещё нет).
func (r *PlanRepository) CreateShare(ctx context.Context, planID uuid.UUID) (*domain.PlanShare, error) {
	id := uuid.New()
	token := uuid.New()
	now := time.Now().UTC()
	row := pgPlanShare{
		ID:        id.String(),
		PlanID:    planID.String(),
		Token:     token.String(),
		CreatedAt: now,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		// Гонка: два параллельных CreateShare на один план — частичный unique (plan_id) WHERE revoked_at IS NULL.
		if isUniqueViolation(err) {
			existing, gerr := r.GetActiveShareByPlanID(ctx, planID)
			if gerr == nil {
				return existing, nil
			}
			return nil, err
		}
		return nil, err
	}
	return &domain.PlanShare{
		ID:        id,
		PlanID:    planID,
		Token:     token,
		CreatedAt: now,
		RevokedAt: nil,
	}, nil
}

// RevokeActiveShareForPlan отзывает активную ссылку плана.
func (r *PlanRepository) RevokeActiveShareForPlan(ctx context.Context, planID uuid.UUID) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&pgPlanShare{}).
		Where("plan_id = ? AND revoked_at IS NULL", planID.String()).
		Update("revoked_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// GetActiveShareByToken возвращает активную share по token.
func (r *PlanRepository) GetActiveShareByToken(ctx context.Context, token uuid.UUID) (*domain.PlanShare, error) {
	var m pgPlanShare
	err := r.db.WithContext(ctx).
		Table("workout_plan_shares").
		Joins("JOIN workout_plans ON workout_plans.id = workout_plan_shares.plan_id").
		Joins("JOIN users ON users.id = workout_plans.user_id").
		Where("workout_plan_shares.token = ? AND workout_plan_shares.revoked_at IS NULL AND users.deleted_at IS NULL", token.String()).
		Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return pgShareToDomain(&m)
}

func (r *PlanRepository) getPlanWithDaysAndExercisesWithDB(db *gorm.DB, planID uuid.UUID) (*domain.Plan, error) {
	var m pgPlan
	err := db.
		Preload("Days", func(d *gorm.DB) *gorm.DB { return d.Order("sort_order") }).
		Preload("Days.Exercises", func(d *gorm.DB) *gorm.DB { return d.Order("sort_order") }).
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

// ClonePlanFromShare выполняет глубокий клон и запись события копирования в одной транзакции.
func (r *PlanRepository) ClonePlanFromShare(ctx context.Context, in repo.CloneFromShareInput) (*domain.Plan, error) {
	var newPlanID uuid.UUID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var shareRow pgPlanShare
		if err := tx.
			Table("workout_plan_shares").
			Joins("JOIN workout_plans ON workout_plans.id = workout_plan_shares.plan_id").
			Joins("JOIN users ON users.id = workout_plans.user_id").
			Where("workout_plan_shares.id = ? AND workout_plan_shares.revoked_at IS NULL AND users.deleted_at IS NULL", in.ShareID.String()).
			Take(&shareRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrNotFound
			}
			return err
		}
		if shareRow.PlanID != in.SourcePlanID.String() {
			return repo.ErrNotFound
		}

		source, err := r.getPlanWithDaysAndExercisesWithDB(tx, in.SourcePlanID)
		if err != nil {
			return err
		}

		newPlanID = uuid.New()
		now := time.Now().UTC()
		srcID := in.SourcePlanID
		newPlan := &domain.Plan{
			ID:           newPlanID,
			UserID:       in.RecipientUserID,
			Name:         in.CopyName,
			IsActive:     false,
			IsPublic:     false,
			Category:     source.Category,
			Level:        source.Level,
			SourcePlanID: &srcID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(domainPlanToPg(newPlan)).Error; err != nil {
			return err
		}

		dayIDMap := make(map[uuid.UUID]uuid.UUID, len(source.Days))
		for _, d := range source.Days {
			newDayID := uuid.New()
			dayIDMap[d.ID] = newDayID
			day := &domain.PlanDay{
				ID:        newDayID,
				PlanID:    newPlanID,
				Name:      d.Name,
				SortOrder: d.SortOrder,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Create(domainPlanDayToPg(day)).Error; err != nil {
				return err
			}
		}

		for _, d := range source.Days {
			newDayID := dayIDMap[d.ID]
			if len(d.Exercises) == 0 {
				continue
			}
			exercises := make([]*domain.PlanDayExercise, 0, len(d.Exercises))
			for i := range d.Exercises {
				ex := d.Exercises[i]
				exercises = append(exercises, &domain.PlanDayExercise{
					DayID:           newDayID,
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
				})
			}
			if err := r.createDayExercisesWithDB(tx, newDayID, exercises); err != nil {
				return err
			}
		}

		srcStr := in.SourcePlanID.String()
		shareStr := in.ShareID.String()
		event := pgPlanCopyEvent{
			ID:              uuid.New().String(),
			SourcePlanID:    &srcStr,
			CopyPlanID:      newPlanID.String(),
			RecipientUserID: in.RecipientUserID.String(),
			Channel:         domain.PlanCopyChannelShare,
			ShareID:         &shareStr,
			CreatedAt:       now,
		}
		return tx.Create(&event).Error
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, newPlanID)
}

// GetShareCopyStats возвращает агрегаты копирований по исходному плану (канал share).
func (r *PlanRepository) GetShareCopyStats(ctx context.Context, sourcePlanID uuid.UUID) (int64, int64, error) {
	var row struct {
		Total            int64 `gorm:"column:total"`
		UniqueRecipients int64 `gorm:"column:unique_recipients"`
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)::bigint AS total,
		       COUNT(DISTINCT recipient_user_id)::bigint AS unique_recipients
		FROM workout_plan_copy_events
		WHERE source_plan_id = ? AND channel = ?
	`, sourcePlanID.String(), domain.PlanCopyChannelShare).Scan(&row).Error
	if err != nil {
		return 0, 0, err
	}
	return row.Total, row.UniqueRecipients, nil
}
