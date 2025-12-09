package interfaces

import (
	"context"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
)

// PasswordResetRepository определяет контракт для работы с кодами сброса пароля.
type PasswordResetRepository interface {
	// Create создает новую запись с кодом сброса пароля.
	Create(ctx context.Context, pr *domain.PasswordReset) error

	// GetActiveByUserID возвращает активную (не истекшую и не использованную) запись по user_id.
	// Возвращает (nil, ErrNotFound), если активного кода нет.
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.PasswordReset, error)

	// GetByID возвращает запись сброса пароля по её ID.
	// Используется для получения обновленного значения попыток после IncrementAttempts.
	GetByID(ctx context.Context, id int64) (*domain.PasswordReset, error)

	// IncrementAttempts увеличивает счетчик попыток для записи по её ID.
	IncrementAttempts(ctx context.Context, id int64) error

	// MarkAsUsed помечает код как использованный, устанавливая used_at в текущее время.
	MarkAsUsed(ctx context.Context, id int64) error

	// DeleteByUserID удаляет все записи кодов сброса пароля для указанного пользователя.
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
