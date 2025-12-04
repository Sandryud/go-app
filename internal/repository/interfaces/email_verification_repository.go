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

	// IncrementAttempts увеличивает счетчик попыток для записи по её ID.
	IncrementAttempts(ctx context.Context, id int64) error

	// DeleteByUserID удаляет все записи кодов для указанного пользователя.
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}


