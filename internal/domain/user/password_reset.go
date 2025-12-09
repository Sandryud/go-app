package user

import (
	"time"

	"github.com/google/uuid"
)

// PasswordReset представляет доменную модель кода сброса пароля.
// Используется для хранения информации о кодах сброса пароля пользователей.
type PasswordReset struct {
	ID          int64      // Идентификатор записи (соответствует BIGSERIAL в БД)
	UserID      uuid.UUID  // Пользователь, для которого создан код
	CodeHash    string     // Хэш одноразового кода сброса пароля
	ExpiresAt   time.Time  // Время истечения кода
	Attempts    int        // Количество использованных попыток ввода кода
	MaxAttempts int        // Максимально допустимое количество попыток
	CreatedAt   time.Time  // Время создания записи
	UsedAt      *time.Time // Время использования кода (nil, если код ещё не использован)
}
