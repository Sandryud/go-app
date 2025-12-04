package verification

import (
	"context"
	"fmt"
	"time"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
	"workout-app/pkg/password"
)

// VerificationResult представляет результат проверки кода.
type VerificationResult int

const (
	VerificationSuccess VerificationResult = iota
	VerificationCodeInvalid
	VerificationAttemptsExceeded
	VerificationExpired
)

// VerifyCode проверяет код подтверждения и обрабатывает попытки.
// Возвращает результат проверки и обновленную запись верификации.
// Исправляет race condition, получая обновленное значение попыток из БД.
func VerifyCode(
	ctx context.Context,
	verification *domain.EmailVerification,
	code string,
	emailVerifs repo.EmailVerificationRepository,
) (VerificationResult, *domain.EmailVerification, error) {
	// Проверяем TTL
	if time.Now().UTC().After(verification.ExpiresAt) {
		return VerificationExpired, nil, nil
	}

	// Сравниваем код по хэшу
	if err := password.Compare(verification.CodeHash, code); err != nil {
		// Увеличиваем количество попыток
		if err := emailVerifs.IncrementAttempts(ctx, verification.ID); err != nil {
			return 0, nil, fmt.Errorf("failed to increment attempts: %w", err)
		}

		// Получаем обновленное значение попыток из БД для исправления race condition
		updatedVerification, err := getVerificationByID(ctx, emailVerifs, verification.ID)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to get updated verification: %w", err)
		}

		// Проверяем, не превышен ли лимит попыток
		if updatedVerification.Attempts >= updatedVerification.MaxAttempts {
			return VerificationAttemptsExceeded, updatedVerification, nil
		}

		return VerificationCodeInvalid, updatedVerification, nil
	}

	// Код верный
	return VerificationSuccess, verification, nil
}

// getVerificationByID получает запись верификации по ID.
// Вспомогательная функция для получения обновленного значения попыток.
func getVerificationByID(ctx context.Context, emailVerifs repo.EmailVerificationRepository, id int64) (*domain.EmailVerification, error) {
	verification, err := emailVerifs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification by ID: %w", err)
	}
	return verification, nil
}
