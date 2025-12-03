package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
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
}

// ProfileUpdateInput описывает допустимые изменения в профиле пользователя
// на уровне бизнес-логики (usecase). Все поля опциональны.
type ProfileUpdateInput struct {
	Email         *string
	Username      *string
	FirstName     *string
	LastName      *string
	BirthDate     *time.Time
	Gender        *string
	AvatarURL     *string
	Role          *domain.Role
	TrainingLevel *domain.TrainingLevel
}

type service struct {
	users repo.UserRepository
}

// NewService создаёт новый сервис пользователей.
func NewService(users repo.UserRepository) Service {
	return &service{users: users}
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
func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, input ProfileUpdateInput) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Применяем изменения к доменной модели
	if input.Email != nil {
		user.Email = *input.Email
	}
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



