package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
)

// pgPasswordReset представляет ORM-модель для таблицы password_resets.
type pgPasswordReset struct {
	ID          int64      `gorm:"column:id;type:bigserial;primaryKey"`
	UserID      string     `gorm:"column:user_id;type:uuid;not null"`
	CodeHash    string     `gorm:"column:code_hash;type:varchar(255);not null"`
	ExpiresAt   time.Time  `gorm:"column:expires_at;type:timestamptz;not null"`
	Attempts    int        `gorm:"column:attempts;type:int;not null"`
	MaxAttempts int        `gorm:"column:max_attempts;type:int;not null"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	UsedAt      *time.Time `gorm:"column:used_at;type:timestamptz"`
}

func (pgPasswordReset) TableName() string {
	return "password_resets"
}

func (m *pgPasswordReset) toDomain() (*domain.PasswordReset, error) {
	userID, err := uuid.Parse(m.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user_id UUID: %w", err)
	}

	return &domain.PasswordReset{
		ID:          m.ID,
		UserID:      userID,
		CodeHash:    m.CodeHash,
		ExpiresAt:   m.ExpiresAt,
		Attempts:    m.Attempts,
		MaxAttempts: m.MaxAttempts,
		CreatedAt:   m.CreatedAt,
		UsedAt:      m.UsedAt,
	}, nil
}

func fromDomainPasswordReset(pr *domain.PasswordReset) *pgPasswordReset {
	return &pgPasswordReset{
		ID:          pr.ID,
		UserID:      pr.UserID.String(),
		CodeHash:    pr.CodeHash,
		ExpiresAt:   pr.ExpiresAt,
		Attempts:    pr.Attempts,
		MaxAttempts: pr.MaxAttempts,
		CreatedAt:   pr.CreatedAt,
		UsedAt:      pr.UsedAt,
	}
}

// PasswordResetRepository реализует repo.PasswordResetRepository на GORM/Postgres.
type PasswordResetRepository struct {
	db *gorm.DB
}

// Убедимся на этапе компиляции, что структура реализует интерфейс.
var _ repo.PasswordResetRepository = (*PasswordResetRepository)(nil)

// NewPasswordResetRepository создает новый репозиторий для кодов сброса пароля.
func NewPasswordResetRepository(db *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

// Create создает новую запись с кодом сброса пароля.
func (r *PasswordResetRepository) Create(ctx context.Context, pr *domain.PasswordReset) error {
	model := fromDomainPasswordReset(pr)
	return r.db.WithContext(ctx).Create(model).Error
}

// GetActiveByUserID возвращает активную (не истекшую и не использованную) запись по user_id.
func (r *PasswordResetRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.PasswordReset, error) {
	var model pgPasswordReset

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND expires_at > NOW() AND used_at IS NULL", userID.String()).
		Order("created_at DESC").
		Take(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	pr, err := model.toDomain()
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// GetByID возвращает запись сброса пароля по её ID.
func (r *PasswordResetRepository) GetByID(ctx context.Context, id int64) (*domain.PasswordReset, error) {
	var model pgPasswordReset

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Take(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	pr, err := model.toDomain()
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// IncrementAttempts увеличивает счетчик попыток для записи по её ID.
func (r *PasswordResetRepository) IncrementAttempts(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).
		Model(&pgPasswordReset{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1"))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// MarkAsUsed помечает код как использованный, устанавливая used_at в текущее время.
func (r *PasswordResetRepository) MarkAsUsed(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&pgPasswordReset{}).
		Where("id = ?", id).
		UpdateColumn("used_at", now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// DeleteByUserID удаляет все записи кодов сброса пароля для указанного пользователя.
func (r *PasswordResetRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID.String()).
		Delete(&pgPasswordReset{})

	if result.Error != nil {
		return result.Error
	}
	return nil
}
