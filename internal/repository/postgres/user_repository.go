package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"gorm.io/gorm"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
)

// pgUser представляет собой ORM-модель для таблицы users.
// Она максимально близко отражает схему БД и маппится в доменную модель User.
type pgUser struct {
	ID            string    `gorm:"column:id;type:uuid;primaryKey"`
	Email         string    `gorm:"column:email;type:varchar(255);not null"`
	PasswordHash  string    `gorm:"column:password_hash;type:varchar(255);not null"`
	Username      string    `gorm:"column:username;type:varchar(50);not null"`
	FirstName     string    `gorm:"column:first_name;type:varchar(100)"`
	LastName      string    `gorm:"column:last_name;type:varchar(100)"`
	BirthDate     *time.Time `gorm:"column:birth_date;type:date"`
	Gender        string    `gorm:"column:gender;type:text"`
	AvatarURL     string    `gorm:"column:avatar_url;type:text"`
	Role          string    `gorm:"column:role;type:text;not null"`
	TrainingLevel string    `gorm:"column:training_level;type:text;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;type:timestamptz"`
}

func (pgUser) TableName() string {
	return "users"
}

// UserRepository реализует repo.UserRepository с использованием GORM и Postgres.
type UserRepository struct {
	db *gorm.DB
}

// Убедимся на этапе компиляции, что структура реализует интерфейс.
var _ repo.UserRepository = (*UserRepository)(nil)

// NewUserRepository создает новый репозиторий пользователей.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// isUniqueViolation проверяет, является ли ошибка нарушением уникального ограничения PostgreSQL.
// Ориентируется на код ошибки 23505 (unique_violation) и, при наличии, имя индекса/constraint.
func isUniqueViolation(err error, constraintNames ...string) bool {
	if err == nil {
		return false
	}

	// Предпочитаем структурированную ошибку драйвера pgx
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != "23505" { // unique_violation
			return false
		}
		// Если конкретные имена не заданы — достаточно кода ошибки
		if len(constraintNames) == 0 {
			return true
		}
		for _, name := range constraintNames {
			if name != "" && strings.EqualFold(pgErr.ConstraintName, name) {
				return true
			}
		}
		return false
	}

	// Fallback для нестандартных ошибок: ищем 23505 и имя индекса/constraint в сообщении
	errStr := err.Error()
	if !strings.Contains(errStr, "23505") {
		return false
	}
	if len(constraintNames) == 0 {
		return true
	}
	lower := strings.ToLower(errStr)
	for _, name := range constraintNames {
		if name != "" && strings.Contains(lower, strings.ToLower(name)) {
			return true
		}
	}
	return false
}

// toDomain маппит ORM-модель в доменную.
func (m *pgUser) toDomain() (*domain.User, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:            id,
		Email:         m.Email,
		PasswordHash:  m.PasswordHash,
		Username:      m.Username,
		FirstName:     m.FirstName,
		LastName:      m.LastName,
		BirthDate:     m.BirthDate,
		Gender:        m.Gender,
		AvatarURL:     m.AvatarURL,
		Role:          domain.Role(m.Role),
		TrainingLevel: domain.TrainingLevel(m.TrainingLevel),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}, nil
}

// fromDomain маппит доменную модель в ORM-модель.
func fromDomain(u *domain.User) *pgUser {
	return &pgUser{
		ID:            u.ID.String(),
		Email:         u.Email,
		PasswordHash:  u.PasswordHash,
		Username:      u.Username,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		BirthDate:     u.BirthDate,
		Gender:        u.Gender,
		AvatarURL:     u.AvatarURL,
		Role:          string(u.Role),
		TrainingLevel: string(u.TrainingLevel),
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		DeletedAt:     u.DeletedAt,
	}
}

// Create создает нового пользователя в БД.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	model := fromDomain(user)
	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		// Проверка на нарушение уникальности email
		if isUniqueViolation(err, "idx_users_email_unique") || strings.Contains(err.Error(), "idx_users_email_unique") {
			return repo.ErrEmailExists
		}
		// Проверка на нарушение уникальности username
		if isUniqueViolation(err, "idx_users_username_unique") || strings.Contains(err.Error(), "idx_users_username_unique") {
			return repo.ErrUsernameExists
		}
		return err
	}
	return nil
}

// oneByCondition возвращает одну запись по условию с учётом soft delete.
func (r *UserRepository) oneByCondition(ctx context.Context, query string, args ...interface{}) (*domain.User, error) {
	var model pgUser
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where(query, args...).
		Take(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.toDomain()
}

// GetByID возвращает пользователя по идентификатору.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.oneByCondition(ctx, "id = ?", id.String())
}

// GetByEmail возвращает пользователя по email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.oneByCondition(ctx, "email = ?", email)
}

// GetByUsername возвращает пользователя по username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.oneByCondition(ctx, "username = ?", username)
}

// List возвращает всех активных (не удалённых) пользователей.
func (r *UserRepository) List(ctx context.Context) ([]*domain.User, error) {
	var models []pgUser
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	users := make([]*domain.User, 0, len(models))
	for i := range models {
		u, err := models[i].toDomain()
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Update обновляет данные пользователя.
// Не обновляет защищенные поля: id, created_at, password_hash.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	model := fromDomain(user)
	
	// Используем выборочное обновление для защиты критичных полей
	updates := map[string]interface{}{
		"email":          model.Email,
		"username":       model.Username,
		"first_name":     model.FirstName,
		"last_name":      model.LastName,
		"birth_date":     model.BirthDate,
		"gender":         model.Gender,
		"avatar_url":     model.AvatarURL,
		"role":           model.Role,
		"training_level": model.TrainingLevel,
		// updated_at обновляется на стороне БД триггером update_users_updated_at
	}
	
	result := r.db.WithContext(ctx).
		Model(&pgUser{}).
		Where("id = ? AND deleted_at IS NULL", model.ID).
		Updates(updates)
	
	if result.Error != nil {
		// Проверка на нарушение уникальности при обновлении
		if isUniqueViolation(result.Error, "idx_users_email_unique") || strings.Contains(result.Error.Error(), "idx_users_email_unique") {
			return repo.ErrEmailExists
		}
		if isUniqueViolation(result.Error, "idx_users_username_unique") || strings.Contains(result.Error.Error(), "idx_users_username_unique") {
			return repo.ErrUsernameExists
		}
		return result.Error
	}

	// Если ни одна строка не была обновлена — пользователя нет или он уже удалён
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	
	return nil
}

// SoftDelete помечает пользователя как удалённого.
// Синхронизировано с доменным методом MarkDeleted (также обновляет updated_at).
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	
	// Обновляем deleted_at и updated_at синхронно с доменной логикой MarkDeleted
	result := r.db.WithContext(ctx).
		Model(&pgUser{}).
		Where("id = ? AND deleted_at IS NULL", id.String()).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		})
	
	if result.Error != nil {
		return result.Error
	}
	
	// Проверяем, была ли обновлена хотя бы одна запись
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	
	return nil
}


