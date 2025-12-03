package interfaces

import (
	"context"
	"errors"

	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
)

// ErrNotFound возвращается, когда сущность не найдена в хранилище.
var ErrNotFound = errors.New("entity not found")

// ErrEmailExists возвращается, когда пользователь с таким email уже существует.
var ErrEmailExists = errors.New("email already exists")

// ErrUsernameExists возвращается, когда пользователь с таким username уже существует.
var ErrUsernameExists = errors.New("username already exists")

// UserRepository определяет контракт для работы с пользователями на уровне хранилища.
//
// Интерфейс оперирует доменной моделью User и не раскрывает деталей реализации (GORM, SQL и т.п.).
type UserRepository interface {
	// Create создает нового пользователя.
	// Возвращает ErrEmailExists, если email уже используется.
	// Возвращает ErrUsernameExists, если username уже используется.
	Create(ctx context.Context, user *domain.User) error

	// GetByID возвращает пользователя по идентификатору.
	// Возвращает (nil, ErrNotFound), если пользователь не найден или мягко удалён.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetByEmail возвращает пользователя по email.
	// Возвращает (nil, ErrNotFound), если пользователь не найден или мягко удалён.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetByUsername возвращает пользователя по username.
	// Возвращает (nil, ErrNotFound), если пользователь не найден или мягко удалён.
	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	// Update обновляет данные пользователя.
	// Не обновляет защищенные поля: id, created_at, password_hash.
	Update(ctx context.Context, user *domain.User) error

	// SoftDelete помечает пользователя как удалённого (soft delete).
	SoftDelete(ctx context.Context, id uuid.UUID) error
}


