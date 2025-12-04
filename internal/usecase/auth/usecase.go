package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
	jwtsvc "workout-app/pkg/jwt"
	"workout-app/pkg/password"
)

// EmailSender описывает контракт для отправки кода подтверждения email.
type EmailSender interface {
	SendEmailVerificationCode(ctx context.Context, email, code string) error
}

// Service описывает usecase-слой, связанный с аутентификацией:
// регистрацию, подтверждение email и логин.
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
}

// Ошибки бизнес-логики usecase-слоя.
var (
	ErrEmailAlreadyVerified         = fmt.Errorf("email already verified")
	ErrVerificationCodeNotFound     = fmt.Errorf("verification code not found")
	ErrVerificationCodeInvalid      = fmt.Errorf("verification code invalid")
	ErrVerificationAttemptsExceeded = fmt.Errorf("verification attempts exceeded")
	ErrEmailNotVerified             = fmt.Errorf("email not verified")
	ErrInvalidCredentials           = fmt.Errorf("invalid email or password")
	ErrInvalidRefreshToken          = fmt.Errorf("invalid refresh token")
)

const verificationCodeLength = 6

type service struct {
	users           repo.UserRepository
	emailVerifs     repo.EmailVerificationRepository
	jwt             jwtsvc.Service
	emailSender     EmailSender
	verificationTTL time.Duration
	maxAttempts     int
}

// NewService создаёт новый auth usecase-сервис.
// verificationTTL задаёт время жизни кода подтверждения,
// maxAttempts — максимальное количество неверных попыток ввода кода.
func NewService(
	users repo.UserRepository,
	emailVerifs repo.EmailVerificationRepository,
	jwt jwtsvc.Service,
	emailSender EmailSender,
	verificationTTL time.Duration,
	maxAttempts int,
) Service {
	return &service{
		users:           users,
		emailVerifs:     emailVerifs,
		jwt:             jwt,
		emailSender:     emailSender,
		verificationTTL: verificationTTL,
		maxAttempts:     maxAttempts,
	}
}

// Register регистрирует нового пользователя и отправляет код подтверждения email.
func (s *service) Register(ctx context.Context, email, rawPassword, username string) (*domain.User, error) {
	if email == "" || rawPassword == "" || username == "" {
		return nil, fmt.Errorf("email, password and username are required")
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

	// Генерируем одноразовый код и его хэш.
	code, err := generateNumericCode(verificationCodeLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}

	codeHash, err := password.Hash(code)
	if err != nil {
		return nil, fmt.Errorf("failed to hash verification code: %w", err)
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
		return nil, err
	}

	if err := s.emailSender.SendEmailVerificationCode(ctx, user.Email, code); err != nil {
		return nil, fmt.Errorf("failed to send verification email: %w", err)
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

	// Проверим TTL на всякий случай (репозиторий уже фильтрует по expires_at > now).
	if time.Now().UTC().After(v.ExpiresAt) {
		_ = s.emailVerifs.DeleteByUserID(ctx, user.ID)
		return nil, "", "", ErrVerificationCodeNotFound
	}

	// Сравниваем код по хэшу.
	if err := password.Compare(v.CodeHash, code); err != nil {
		newAttempts := v.Attempts + 1

		// Увеличиваем количество попыток.
		_ = s.emailVerifs.IncrementAttempts(ctx, v.ID)

		if newAttempts >= v.MaxAttempts {
			_ = s.emailVerifs.DeleteByUserID(ctx, user.ID)
			return nil, "", "", ErrVerificationAttemptsExceeded
		}

		return nil, "", "", ErrVerificationCodeInvalid
	}

	// Успешное подтверждение: отмечаем email как подтверждённый.
	user.IsEmailVerified = true
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, "", "", err
	}

	// Удаляем все коды для пользователя.
	_ = s.emailVerifs.DeleteByUserID(ctx, user.ID)

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

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == repo.ErrNotFound {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", err
	}

	if err := password.Compare(user.PasswordHash, rawPassword); err != nil {
		return nil, "", "", ErrInvalidCredentials
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

// generateNumericCode генерирует криптографически стойкий числовой код заданной длины.
func generateNumericCode(length int) (string, error) {
	const digits = "0123456789"

	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	code := make([]byte, length)
	max := big.NewInt(int64(len(digits)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = digits[n.Int64()]
	}

	return string(code), nil
}
