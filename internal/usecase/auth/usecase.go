package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
	jwtsvc "workout-app/pkg/jwt"
	"workout-app/pkg/mailer"
	"workout-app/pkg/password"
	"workout-app/pkg/verification"
)

// Service описывает usecase-слой, связанный с аутентификацией:
// регистрацию, подтверждение email, логин и refresh токенов.
type Service interface {
	// Register регистрирует пользователя, создаёт код подтверждения email и отправляет его.
	// Возвращает созданного пользователя (без токенов).
	Register(ctx context.Context, email, password, username string) (*domain.User, error)

	// VerifyEmail проверяет код подтверждения email, активирует пользователя
	// и возвращает пользователя с парой access/refresh токенов.
	VerifyEmail(ctx context.Context, email, code string) (*domain.User, string, string, error)

	// Login выполняет вход по email/паролю, проверяя, что email подтверждён.
	// Возвращает пользователя и пару access/refresh токенов.
	Login(ctx context.Context, email, password string) (*domain.User, string, string, error)

	// Refresh обновляет пару access/refresh токенов по действительному refresh-токену.
	Refresh(ctx context.Context, refreshToken string) (*domain.User, string, string, error)

	// ResendVerificationCode повторно отправляет код подтверждения email,
	// если аккаунт существует и ещё не подтверждён.
	ResendVerificationCode(ctx context.Context, email string) error

	// RestoreAccount восстанавливает удалённый аккаунт по email и паролю.
	// Возвращает пользователя и пару access/refresh токенов.
	// Требует, чтобы email был подтверждён.
	RestoreAccount(ctx context.Context, email, password string) (*domain.User, string, string, error)

	// RequestPasswordReset запрашивает код сброса пароля для указанного email.
	// Отправляет код на email, если пользователь существует, не удалён и email подтверждён.
	RequestPasswordReset(ctx context.Context, email string) error

	// ResetPassword сбрасывает пароль пользователя по коду.
	// Проверяет код, обновляет пароль и помечает код как использованный.
	ResetPassword(ctx context.Context, email, code, newPassword string) error
}

// Ошибки бизнес-логики usecase-слоя.
var (
	ErrEmailAlreadyVerified          = fmt.Errorf("email already verified")
	ErrVerificationCodeNotFound      = fmt.Errorf("verification code not found")
	ErrVerificationCodeInvalid       = fmt.Errorf("verification code invalid")
	ErrVerificationAttemptsExceeded  = fmt.Errorf("verification attempts exceeded")
	ErrEmailNotVerified              = fmt.Errorf("email not verified")
	ErrInvalidCredentials            = fmt.Errorf("invalid email or password")
	ErrInvalidRefreshToken           = fmt.Errorf("invalid refresh token")
	ErrEmailUnverifiedExists         = fmt.Errorf("unverified account with this email already exists")
	ErrAccountDeleted                = fmt.Errorf("account is deleted")
	ErrAccountNotDeleted             = fmt.Errorf("account is not deleted")
	ErrPasswordResetCodeNotFound     = fmt.Errorf("password reset code not found")
	ErrPasswordResetCodeInvalid      = fmt.Errorf("password reset code invalid")
	ErrPasswordResetAttemptsExceeded = fmt.Errorf("password reset attempts exceeded")
	ErrPasswordResetCodeUsed         = fmt.Errorf("password reset code already used")
)

type service struct {
	users           repo.UserRepository
	emailVerifs     repo.EmailVerificationRepository
	passwordResets  repo.PasswordResetRepository
	jwt             jwtsvc.Service
	emailSender     mailer.EmailSender
	verificationTTL time.Duration
	maxAttempts     int
	codeLength      int
}

// NewService создаёт новый auth usecase-сервис.
// verificationTTL задаёт время жизни кода подтверждения,
// maxAttempts — максимальное количество неверных попыток ввода кода.
func NewService(
	users repo.UserRepository,
	emailVerifs repo.EmailVerificationRepository,
	passwordResets repo.PasswordResetRepository,
	jwt jwtsvc.Service,
	emailSender mailer.EmailSender,
	verificationTTL time.Duration,
	maxAttempts int,
	codeLength int,
) Service {
	return &service{
		users:           users,
		emailVerifs:     emailVerifs,
		passwordResets:  passwordResets,
		jwt:             jwt,
		emailSender:     emailSender,
		verificationTTL: verificationTTL,
		maxAttempts:     maxAttempts,
		codeLength:      codeLength,
	}
}

// Register регистрирует нового пользователя и отправляет код подтверждения email.
func (s *service) Register(ctx context.Context, email, rawPassword, username string) (*domain.User, error) {
	if email == "" || rawPassword == "" || username == "" {
		return nil, fmt.Errorf("email, password and username are required")
	}

	// Проверяем, существует ли пользователь (включая удалённых)
	existing, err := s.users.GetByEmailIncludingDeleted(ctx, email)
	if err != nil {
		if err != repo.ErrNotFound {
			return nil, err
		}
		// err == repo.ErrNotFound - продолжаем создание нового пользователя
	} else {
		// Пользователь найден
		if existing.IsDeleted() {
			// Аккаунт удалён - предлагаем восстановление
			return nil, ErrAccountDeleted
		}
		if existing.IsEmailVerified {
			// Обычный конфликт: подтверждённый email.
			return nil, repo.ErrEmailExists
		}
		// Email уже существует, но не подтверждён.
		return nil, ErrEmailUnverifiedExists
	}

	// Хешируем пароль на уровне usecase.
	hashed, err := password.Hash(rawPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := domain.NewUser(email, hashed, username)
	user.IsEmailVerified = false

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	if err := s.createAndSendVerificationCode(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// VerifyEmail подтверждает email по коду, активирует пользователя
// и возвращает пару access/refresh токенов.
func (s *service) VerifyEmail(ctx context.Context, email, code string) (*domain.User, string, string, error) {
	if email == "" || code == "" {
		return nil, "", "", fmt.Errorf("email and code are required")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Если пользователь не найден, возвращаем ErrVerificationCodeNotFound
		// для единообразия с ошибкой отсутствия кода верификации.
		// Это отличается от ResendVerificationCode, где ErrNotFound игнорируется
		// для безопасности (чтобы не раскрывать существование аккаунта).
		// Здесь же пользователь явно пытается верифицировать код, поэтому
		// возвращаем ошибку для ясности. Это не раскрывает дополнительной информации,
		// так как отсутствие пользователя или кода верификации приводит к одинаковому
		// результату - невозможности верификации.
		if err == repo.ErrNotFound {
			return nil, "", "", ErrVerificationCodeNotFound
		}
		return nil, "", "", err
	}

	if user.IsEmailVerified {
		return nil, "", "", ErrEmailAlreadyVerified
	}

	v, err := s.emailVerifs.GetActiveByUserID(ctx, user.ID)
	if err != nil {
		if err == repo.ErrNotFound {
			return nil, "", "", ErrVerificationCodeNotFound
		}
		return nil, "", "", err
	}

	// Используем общую функцию проверки кода
	result, _, err := verification.VerifyCode(ctx, v, code, s.emailVerifs)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to verify code: %w", err)
	}

	switch result {
	case verification.VerificationExpired:
		// Игнорируем ErrNotFound, так как запись могла быть уже удалена
		// (например, другим запросом или по истечении TTL)
		if err := s.emailVerifs.DeleteByUserID(ctx, user.ID); err != nil && err != repo.ErrNotFound {
			return nil, "", "", fmt.Errorf("failed to delete expired verification: %w", err)
		}
		return nil, "", "", ErrVerificationCodeNotFound
	case verification.VerificationAttemptsExceeded:
		// Игнорируем ErrNotFound, так как запись могла быть уже удалена
		// (например, другим запросом или по истечении TTL)
		if err := s.emailVerifs.DeleteByUserID(ctx, user.ID); err != nil && err != repo.ErrNotFound {
			return nil, "", "", fmt.Errorf("failed to delete verification after exceeded attempts: %w", err)
		}
		return nil, "", "", ErrVerificationAttemptsExceeded
	case verification.VerificationCodeInvalid:
		return nil, "", "", ErrVerificationCodeInvalid
	case verification.VerificationSuccess:
		// Продолжаем обработку успешной верификации
	default:
		return nil, "", "", fmt.Errorf("unknown verification result: %d", result)
	}

	// Успешное подтверждение: отмечаем email как подтверждённый.
	user.IsEmailVerified = true
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, "", "", err
	}

	// Удаляем все коды для пользователя.
	if err := s.emailVerifs.DeleteByUserID(ctx, user.ID); err != nil {
		return nil, "", "", fmt.Errorf("failed to delete verification codes: %w", err)
	}

	// Генерируем access/refresh токены.
	access, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return nil, "", "", err
	}

	refresh, _, err := s.jwt.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", "", err
	}

	return user, access, refresh, nil
}

// Login выполняет вход по email/паролю и проверяет, что email подтверждён.
func (s *service) Login(ctx context.Context, email, rawPassword string) (*domain.User, string, string, error) {
	if email == "" || rawPassword == "" {
		return nil, "", "", fmt.Errorf("email and password are required")
	}

	// Используем метод, который находит пользователя включая удалённых
	user, err := s.users.GetByEmailIncludingDeleted(ctx, email)
	if err != nil {
		if err == repo.ErrNotFound {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", err
	}

	// Проверяем пароль
	if err := password.Compare(user.PasswordHash, rawPassword); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	// Проверяем, удалён ли аккаунт
	if user.IsDeleted() {
		return nil, "", "", ErrAccountDeleted
	}

	if !user.IsEmailVerified {
		return nil, "", "", ErrEmailNotVerified
	}

	access, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return nil, "", "", err
	}

	refresh, _, err := s.jwt.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", "", err
	}

	return user, access, refresh, nil
}

// Refresh обновляет пару access/refresh токенов по действительному refresh-токену.
func (s *service) Refresh(ctx context.Context, refreshToken string) (*domain.User, string, string, error) {
	if refreshToken == "" {
		return nil, "", "", fmt.Errorf("refresh token is required")
	}

	claims, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, "", "", ErrInvalidRefreshToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, "", "", ErrInvalidRefreshToken
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == repo.ErrNotFound {
			return nil, "", "", ErrInvalidRefreshToken
		}
		return nil, "", "", err
	}

	// Не выдаём новые токены для мягко удалённых пользователей.
	if user.IsDeleted() {
		return nil, "", "", ErrInvalidRefreshToken
	}

	// Не выдаём новые токены, если email не подтверждён.
	if !user.IsEmailVerified {
		return nil, "", "", ErrEmailNotVerified
	}

	access, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return nil, "", "", err
	}

	refresh, _, err := s.jwt.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", "", err
	}

	return user, access, refresh, nil
}

// ResendVerificationCode повторно отправляет код подтверждения email,
// если аккаунт существует и ещё не подтверждён.
func (s *service) ResendVerificationCode(ctx context.Context, email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == repo.ErrNotFound {
			// Не раскрываем, что пользователя нет — считаем успешным no-op.
			return nil
		}
		return err
	}

	if user.IsEmailVerified {
		// Уже подтверждён — handler решит, что ответить клиенту.
		return ErrEmailAlreadyVerified
	}

	// Удаляем все старые коды для пользователя (если есть).
	if err := s.emailVerifs.DeleteByUserID(ctx, user.ID); err != nil && err != repo.ErrNotFound {
		return err
	}

	return s.createAndSendVerificationCode(ctx, user)
}

// createAndSendVerificationCode создаёт запись с кодом подтверждения email
// и отправляет его пользователю.
func (s *service) createAndSendVerificationCode(ctx context.Context, user *domain.User) error {
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
	}

	if err := s.emailVerifs.Create(ctx, verification); err != nil {
		return err
	}

	if err := s.emailSender.SendEmailVerificationCode(ctx, user.Email, code); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

// RestoreAccount восстанавливает удалённый аккаунт по email и паролю.
// Возвращает пользователя и пару access/refresh токенов.
// Требует, чтобы email был подтверждён.
func (s *service) RestoreAccount(ctx context.Context, email, rawPassword string) (*domain.User, string, string, error) {
	if email == "" || rawPassword == "" {
		return nil, "", "", fmt.Errorf("email and password are required")
	}

	// Находим пользователя включая удалённых
	user, err := s.users.GetByEmailIncludingDeleted(ctx, email)
	if err != nil {
		if err == repo.ErrNotFound {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", err
	}

	// Проверяем пароль
	if err := password.Compare(user.PasswordHash, rawPassword); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	// Проверяем, что аккаунт действительно удалён
	if !user.IsDeleted() {
		return nil, "", "", ErrAccountNotDeleted
	}

	// Восстанавливаем аккаунт
	if err := s.users.RestoreAccount(ctx, user.ID); err != nil {
		return nil, "", "", fmt.Errorf("failed to restore account: %w", err)
	}

	// Перезагружаем пользователя из БД для синхронизации доменной модели
	restoredUser, err := s.users.GetByID(ctx, user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get restored user: %w", err)
	}

	// Проверяем, что email подтверждён (необходимо для генерации токенов)
	if !restoredUser.IsEmailVerified {
		return nil, "", "", ErrEmailNotVerified
	}

	// Генерируем токены
	access, err := s.jwt.GenerateAccessToken(restoredUser)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refresh, _, err := s.jwt.GenerateRefreshToken(restoredUser)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return restoredUser, access, refresh, nil
}

// RequestPasswordReset запрашивает код сброса пароля для указанного email.
func (s *service) RequestPasswordReset(ctx context.Context, email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	// Получаем пользователя (только активных, не удаленных)
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == repo.ErrNotFound {
			// Не раскрываем, что пользователя нет — считаем успешным no-op для безопасности.
			return nil
		}
		return err
	}

	// Проверяем, что email подтверждён
	if !user.IsEmailVerified {
		return ErrEmailNotVerified
	}

	// Удаляем все старые коды сброса пароля для пользователя (если есть).
	if err := s.passwordResets.DeleteByUserID(ctx, user.ID); err != nil && err != repo.ErrNotFound {
		return err
	}

	// Создаём и отправляем код сброса пароля
	return s.createAndSendPasswordResetCode(ctx, user)
}

// ResetPassword сбрасывает пароль пользователя по коду.
func (s *service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if email == "" || code == "" || newPassword == "" {
		return fmt.Errorf("email, code and new password are required")
	}

	// Получаем пользователя
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == repo.ErrNotFound {
			return ErrPasswordResetCodeNotFound
		}
		return err
	}

	// Получаем активный код сброса пароля
	pr, err := s.passwordResets.GetActiveByUserID(ctx, user.ID)
	if err != nil {
		if err == repo.ErrNotFound {
			return ErrPasswordResetCodeNotFound
		}
		return err
	}

	// Проверяем код
	result, _, err := s.verifyPasswordResetCode(ctx, pr, code)
	if err != nil {
		return fmt.Errorf("failed to verify password reset code: %w", err)
	}

	switch result {
	case verification.VerificationExpired:
		if err := s.passwordResets.DeleteByUserID(ctx, user.ID); err != nil && err != repo.ErrNotFound {
			return fmt.Errorf("failed to delete expired password reset code: %w", err)
		}
		return ErrPasswordResetCodeNotFound
	case verification.VerificationAttemptsExceeded:
		if err := s.passwordResets.DeleteByUserID(ctx, user.ID); err != nil && err != repo.ErrNotFound {
			return fmt.Errorf("failed to delete password reset code after exceeded attempts: %w", err)
		}
		return ErrPasswordResetAttemptsExceeded
	case verification.VerificationCodeInvalid:
		return ErrPasswordResetCodeInvalid
	case verification.VerificationSuccess:
		// Продолжаем обработку успешной проверки
	default:
		return fmt.Errorf("unknown verification result: %d", result)
	}

	// Хешируем новый пароль
	hashedPassword, err := password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Обновляем пароль пользователя
	if err := s.users.UpdatePassword(ctx, user.ID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Удаляем все коды сброса пароля для пользователя
	if err := s.passwordResets.DeleteByUserID(ctx, user.ID); err != nil && err != repo.ErrNotFound {
		return fmt.Errorf("failed to delete password reset codes: %w", err)
	}

	return nil
}

// createAndSendPasswordResetCode создаёт запись с кодом сброса пароля
// и отправляет его пользователю.
func (s *service) createAndSendPasswordResetCode(ctx context.Context, user *domain.User) error {
	code, err := verification.GenerateNumericCode(s.codeLength)
	if err != nil {
		return fmt.Errorf("failed to generate password reset code: %w", err)
	}

	codeHash, err := password.Hash(code)
	if err != nil {
		return fmt.Errorf("failed to hash password reset code: %w", err)
	}

	now := time.Now().UTC()
	passwordReset := &domain.PasswordReset{
		UserID:      user.ID,
		CodeHash:    codeHash,
		ExpiresAt:   now.Add(s.verificationTTL),
		Attempts:    0,
		MaxAttempts: s.maxAttempts,
		CreatedAt:   now,
	}

	if err := s.passwordResets.Create(ctx, passwordReset); err != nil {
		return err
	}

	if err := s.emailSender.SendPasswordResetCode(ctx, user.Email, code); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	return nil
}

// verifyPasswordResetCode проверяет код сброса пароля и обрабатывает попытки.
// Возвращает результат проверки и обновленную запись сброса пароля.
func (s *service) verifyPasswordResetCode(
	ctx context.Context,
	pr *domain.PasswordReset,
	code string,
) (verification.VerificationResult, *domain.PasswordReset, error) {
	// Проверяем TTL
	if time.Now().UTC().After(pr.ExpiresAt) {
		return verification.VerificationExpired, nil, nil
	}

	// Сравниваем код по хэшу
	if err := password.Compare(pr.CodeHash, code); err != nil {
		// Увеличиваем количество попыток
		if err := s.passwordResets.IncrementAttempts(ctx, pr.ID); err != nil {
			return 0, nil, fmt.Errorf("failed to increment attempts: %w", err)
		}

		// Получаем обновленное значение попыток из БД для исправления race condition
		updatedPR, err := s.passwordResets.GetByID(ctx, pr.ID)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to get updated password reset: %w", err)
		}

		// Проверяем, не превышен ли лимит попыток
		if updatedPR.Attempts >= updatedPR.MaxAttempts {
			return verification.VerificationAttemptsExceeded, updatedPR, nil
		}

		return verification.VerificationCodeInvalid, updatedPR, nil
	}

	// Код верный
	return verification.VerificationSuccess, pr, nil
}
