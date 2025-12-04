package interfaces

import (
	"context"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
)

// EmailVerificationRepository определяет контракт для работы с кодами подтверждения email.
type EmailVerificationRepository interface {
	// Create создает новую запись с кодом подтверждения email.
	Create(ctx context.Context, v *domain.EmailVerification) error

	// GetActiveByUserID возвращает активную (не истекшую) запись по user_id.
	// Возвращает (nil, ErrNotFound), если активного кода нет.
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailVerification, error)

	// GetActiveByUserIDAndNewEmail возвращает активную (не истекшую) запись по user_id и new_email.
	// Используется для поиска кода подтверждения изменения email.
	// Возвращает (nil, ErrNotFound), если активного кода нет.
	GetActiveByUserIDAndNewEmail(ctx context.Context, userID uuid.UUID, newEmail string) (*domain.EmailVerification, error)

	// GetActiveEmailChangeByUserID возвращает активную (не истекшую) запись изменения email по user_id.
	// Используется для поиска кода подтверждения изменения email без знания нового email.
	// Возвращает (nil, ErrNotFound), если активного кода нет.
	GetActiveEmailChangeByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailVerification, error)

	// GetByID возвращает запись верификации по её ID.
	// Используется для получения обновленного значения попыток после IncrementAttempts.
	GetByID(ctx context.Context, id int64) (*domain.EmailVerification, error)

	// IncrementAttempts увеличивает счетчик попыток для записи по её ID.
	IncrementAttempts(ctx context.Context, id int64) error

	// DeleteByUserID удаляет все записи кодов для указанного пользователя.
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteEmailChangeByUserID удаляет все записи кодов изменения email для указанного пользователя.
	DeleteEmailChangeByUserID(ctx context.Context, userID uuid.UUID) error
}
