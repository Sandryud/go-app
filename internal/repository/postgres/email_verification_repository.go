package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
)

// pgEmailVerification представляет ORM-модель для таблицы email_verifications.
type pgEmailVerification struct {
	ID          int64     `gorm:"column:id;type:bigserial;primaryKey"`
	UserID      string    `gorm:"column:user_id;type:uuid;not null"`
	CodeHash    string    `gorm:"column:code_hash;type:varchar(255);not null"`
	ExpiresAt   time.Time `gorm:"column:expires_at;type:timestamptz;not null"`
	Attempts    int       `gorm:"column:attempts;type:int;not null"`
	MaxAttempts int       `gorm:"column:max_attempts;type:int;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null"`
	NewEmail    *string   `gorm:"column:new_email;type:varchar(255)"`
}

func (pgEmailVerification) TableName() string {
	return "email_verifications"
}

func (m *pgEmailVerification) toDomain() (*domain.EmailVerification, error) {
	userID, err := uuid.Parse(m.UserID)
	if err != nil {
		return nil, err
	}

	return &domain.EmailVerification{
		ID:          m.ID,
		UserID:      userID,
		CodeHash:    m.CodeHash,
		ExpiresAt:   m.ExpiresAt,
		Attempts:    m.Attempts,
		MaxAttempts: m.MaxAttempts,
		CreatedAt:   m.CreatedAt,
		NewEmail:    m.NewEmail,
	}, nil
}

func fromDomainEmailVerification(v *domain.EmailVerification) *pgEmailVerification {
	return &pgEmailVerification{
		ID:          v.ID,
		UserID:      v.UserID.String(),
		CodeHash:    v.CodeHash,
		ExpiresAt:   v.ExpiresAt,
		Attempts:    v.Attempts,
		MaxAttempts: v.MaxAttempts,
		CreatedAt:   v.CreatedAt,
		NewEmail:    v.NewEmail,
	}
}

// EmailVerificationRepository реализует repo.EmailVerificationRepository на GORM/Postgres.
type EmailVerificationRepository struct {
	db *gorm.DB
}

// Убедимся на этапе компиляции, что структура реализует интерфейс.
var _ repo.EmailVerificationRepository = (*EmailVerificationRepository)(nil)

// NewEmailVerificationRepository создает новый репозиторий для кодов подтверждения email.
func NewEmailVerificationRepository(db *gorm.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

// Create создает новую запись с кодом подтверждения email.
func (r *EmailVerificationRepository) Create(ctx context.Context, v *domain.EmailVerification) error {
	model := fromDomainEmailVerification(v)
	return r.db.WithContext(ctx).Create(model).Error
}

// GetActiveByUserID возвращает активную (не истекшую) запись по user_id.
func (r *EmailVerificationRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailVerification, error) {
	var model pgEmailVerification

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND expires_at > NOW() AND new_email IS NULL", userID.String()).
		Order("created_at DESC").
		Take(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	return model.toDomain()
}

// GetActiveByUserIDAndNewEmail возвращает активную (не истекшую) запись по user_id и new_email.
func (r *EmailVerificationRepository) GetActiveByUserIDAndNewEmail(ctx context.Context, userID uuid.UUID, newEmail string) (*domain.EmailVerification, error) {
	if newEmail == "" {
		return nil, fmt.Errorf("newEmail cannot be empty")
	}

	var model pgEmailVerification

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND new_email = ? AND expires_at > NOW()", userID.String(), newEmail).
		Order("created_at DESC").
		Take(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	return model.toDomain()
}

// GetActiveEmailChangeByUserID возвращает активную (не истекшую) запись изменения email по user_id.
func (r *EmailVerificationRepository) GetActiveEmailChangeByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailVerification, error) {
	var model pgEmailVerification

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND new_email IS NOT NULL AND expires_at > NOW()", userID.String()).
		Order("created_at DESC").
		Take(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	return model.toDomain()
}

// GetByID возвращает запись верификации по её ID.
func (r *EmailVerificationRepository) GetByID(ctx context.Context, id int64) (*domain.EmailVerification, error) {
	var model pgEmailVerification

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Take(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	return model.toDomain()
}

// IncrementAttempts увеличивает счетчик попыток для записи по её ID.
func (r *EmailVerificationRepository) IncrementAttempts(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).
		Model(&pgEmailVerification{}).
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

// DeleteByUserID удаляет все записи кодов для указанного пользователя.
func (r *EmailVerificationRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID.String()).
		Delete(&pgEmailVerification{})

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeleteEmailChangeByUserID удаляет все записи кодов изменения email для указанного пользователя.
func (r *EmailVerificationRepository) DeleteEmailChangeByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND new_email IS NOT NULL", userID.String()).
		Delete(&pgEmailVerification{})

	if result.Error != nil {
		return result.Error
	}
	return nil
}
