package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
	"workout-app/pkg/mailer"
	"workout-app/pkg/password"
	"workout-app/pkg/verification"
)

// Service описывает usecase-слой для работы с пользователем:
// регистрацию, получение/обновление профиля и мягкое удаление аккаунта.
type Service interface {
	// Register регистрирует нового пользователя на основе минимального контракта:
	// email, хэш пароля, username. Валидация и хеширование выполняются выше (на уровне хендлера/другого usecase).
	// Возвращает созданного пользователя или ошибку (включая ErrEmailExists/ErrUsernameExists).
	Register(ctx context.Context, email, passwordHash, username string) (*domain.User, error)

	// GetByID возвращает пользователя по идентификатору.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetProfile возвращает профиль текущего пользователя (по его ID).
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error)

	// UpdateProfile обновляет профиль пользователя (без изменения пароля).
	UpdateProfile(ctx context.Context, userID uuid.UUID, input ProfileUpdateInput) (*domain.User, error)

	// DeleteAccount выполняет мягкое удаление аккаунта.
	DeleteAccount(ctx context.Context, userID uuid.UUID) error

	// ListUsers возвращает список всех активных пользователей.
	// Предназначено для административных сценариев.
	ListUsers(ctx context.Context) ([]*domain.User, error)

	// RequestEmailChange запрашивает изменение email пользователя.
	// Отправляет код подтверждения на новый email.
	RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) error

	// VerifyEmailChange подтверждает изменение email по коду.
	// Обновляет email пользователя и устанавливает IsEmailVerified = true.
	VerifyEmailChange(ctx context.Context, userID uuid.UUID, code string) (*domain.User, error)
}

// ProfileUpdateInput описывает допустимые изменения в профиле пользователя
// на уровне бизнес-логики (usecase). Все поля опциональны.
// Email нельзя изменить через этот метод, используйте RequestEmailChange и VerifyEmailChange.
type ProfileUpdateInput struct {
	Username      *string
	FirstName     *string
	LastName      *string
	BirthDate     *time.Time
	Gender        *string
	AvatarURL     *string
	Role          *domain.Role
	TrainingLevel *domain.TrainingLevel
}

// Ошибки бизнес-логики usecase-слоя.
var (
	ErrEmailSameAsCurrent           = fmt.Errorf("new email is the same as current email")
	ErrVerificationCodeNotFound     = fmt.Errorf("verification code not found")
	ErrVerificationCodeInvalid      = fmt.Errorf("verification code invalid")
	ErrVerificationAttemptsExceeded = fmt.Errorf("verification attempts exceeded")
)

type service struct {
	users           repo.UserRepository
	emailVerifs     repo.EmailVerificationRepository
	emailSender     mailer.EmailSender
	verificationTTL time.Duration
	maxAttempts     int
	codeLength      int
}

// NewService создаёт новый сервис пользователей.
func NewService(
	users repo.UserRepository,
	emailVerifs repo.EmailVerificationRepository,
	emailSender mailer.EmailSender,
	verificationTTL time.Duration,
	maxAttempts int,
	codeLength int,
) Service {
	return &service{
		users:           users,
		emailVerifs:     emailVerifs,
		emailSender:     emailSender,
		verificationTTL: verificationTTL,
		maxAttempts:     maxAttempts,
		codeLength:      codeLength,
	}
}

// Register регистрирует нового пользователя.
func (s *service) Register(ctx context.Context, email, passwordHash, username string) (*domain.User, error) {
	if email == "" || passwordHash == "" || username == "" {
		return nil, fmt.Errorf("email, passwordHash и username обязательны")
	}

	user := domain.NewUser(email, passwordHash, username)

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID возвращает пользователя по ID.
func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// GetProfile возвращает профиль пользователя.
func (s *service) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

// UpdateProfile обновляет профиль пользователя.
// Email нельзя изменить через этот метод, используйте RequestEmailChange и VerifyEmailChange.
func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, input ProfileUpdateInput) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Применяем изменения к доменной модели
	if input.Username != nil {
		user.Username = *input.Username
	}
	if input.FirstName != nil {
		user.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		user.LastName = *input.LastName
	}
	if input.BirthDate != nil {
		user.BirthDate = input.BirthDate
	}
	if input.Gender != nil {
		user.Gender = *input.Gender
	}
	if input.AvatarURL != nil {
		user.AvatarURL = *input.AvatarURL
	}
	if input.Role != nil {
		user.Role = *input.Role
	}
	if input.TrainingLevel != nil {
		user.TrainingLevel = *input.TrainingLevel
	}

	// Обновляем пользователя в хранилище
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteAccount выполняет мягкое удаление аккаунта.
func (s *service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	return s.users.SoftDelete(ctx, userID)
}

// ListUsers возвращает всех активных пользователей.
func (s *service) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return s.users.List(ctx)
}

// RequestEmailChange запрашивает изменение email пользователя.
// Отправляет код подтверждения на новый email.
func (s *service) RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) error {
	if newEmail == "" {
		return fmt.Errorf("newEmail is required")
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Проверяем, что новый email отличается от текущего
	if user.Email == newEmail {
		return ErrEmailSameAsCurrent
	}

	// Проверяем, что новый email не занят другим пользователем
	existingUser, err := s.users.GetByEmail(ctx, newEmail)
	if err != nil && err != repo.ErrNotFound {
		return err
	}
	if err == nil && existingUser.ID != userID {
		return repo.ErrEmailExists
	}

	// Удаляем старые коды изменения email для этого пользователя
	if err := s.emailVerifs.DeleteEmailChangeByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete old email change codes: %w", err)
	}

	// Создаём и отправляем код подтверждения
	return s.createAndSendEmailChangeCode(ctx, user, newEmail)
}

// VerifyEmailChange подтверждает изменение email по коду.
// Обновляет email пользователя и устанавливает IsEmailVerified = true.
func (s *service) VerifyEmailChange(ctx context.Context, userID uuid.UUID, code string) (*domain.User, error) {
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Находим активный код изменения email
	v, err := s.emailVerifs.GetActiveEmailChangeByUserID(ctx, userID)
	if err != nil {
		if err == repo.ErrNotFound {
			return nil, ErrVerificationCodeNotFound
		}
		return nil, err
	}

	// Проверяем, что newEmail установлен
	if v.NewEmail == nil {
		return nil, fmt.Errorf("verification code is not for email change")
	}

	// Используем общую функцию проверки кода
	result, updatedVerification, err := verification.VerifyCode(ctx, v, code, s.emailVerifs)
	if err != nil {
		return nil, fmt.Errorf("failed to verify code: %w", err)
	}

	switch result {
	case verification.VerificationExpired:
		if err := s.emailVerifs.DeleteEmailChangeByUserID(ctx, userID); err != nil {
			return nil, fmt.Errorf("failed to delete expired verification: %w", err)
		}
		return nil, ErrVerificationCodeNotFound
	case verification.VerificationAttemptsExceeded:
		if err := s.emailVerifs.DeleteEmailChangeByUserID(ctx, userID); err != nil {
			return nil, fmt.Errorf("failed to delete verification after exceeded attempts: %w", err)
		}
		return nil, ErrVerificationAttemptsExceeded
	case verification.VerificationCodeInvalid:
		return nil, ErrVerificationCodeInvalid
	case verification.VerificationSuccess:
		// Продолжаем обработку успешной верификации
	default:
		return nil, fmt.Errorf("unknown verification result: %d", result)
	}

	// Проверяем, что новый email всё ещё не занят
	existingUser, err := s.users.GetByEmail(ctx, *updatedVerification.NewEmail)
	if err != nil && err != repo.ErrNotFound {
		return nil, fmt.Errorf("failed to check email availability: %w", err)
	}
	if err == nil && existingUser.ID != userID {
		// Email занят другим пользователем
		if err := s.emailVerifs.DeleteEmailChangeByUserID(ctx, userID); err != nil {
			return nil, fmt.Errorf("failed to delete verification after email conflict: %w", err)
		}
		return nil, repo.ErrEmailExists
	}

	// Успешное подтверждение: обновляем email пользователя
	user.Email = *updatedVerification.NewEmail
	user.IsEmailVerified = true
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user email: %w", err)
	}

	// Удаляем коды изменения email для пользователя
	if err := s.emailVerifs.DeleteEmailChangeByUserID(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to delete verification codes: %w", err)
	}

	return user, nil
}

// createAndSendEmailChangeCode создаёт запись с кодом подтверждения изменения email
// и отправляет его на новый email.
func (s *service) createAndSendEmailChangeCode(ctx context.Context, user *domain.User, newEmail string) error {
	code, err := verification.GenerateNumericCode(s.codeLength)
	if err != nil {
		return fmt.Errorf("failed to generate verification code: %w", err)
	}

	codeHash, err := password.Hash(code)
	if err != nil {
		return fmt.Errorf("failed to hash verification code: %w", err)
	}

	now := time.Now().UTC()
	verification := &domain.EmailVerification{
		UserID:      user.ID,
		CodeHash:    codeHash,
		ExpiresAt:   now.Add(s.verificationTTL),
		Attempts:    0,
		MaxAttempts: s.maxAttempts,
		CreatedAt:   now,
		NewEmail:    &newEmail,
	}

	if err := s.emailVerifs.Create(ctx, verification); err != nil {
		return fmt.Errorf("failed to create verification code: %w", err)
	}

	if err := s.emailSender.SendEmailVerificationCode(ctx, newEmail, code); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}
