package user

import (
	"time"

	"github.com/google/uuid"
)

// TrainingLevel описывает уровень подготовки пользователя.
type TrainingLevel string

const (
	TrainingLevelBeginner     TrainingLevel = "beginner"     // начинающий
	TrainingLevelIntermediate TrainingLevel = "intermediate" // продолжающий
	TrainingLevelAdvanced     TrainingLevel = "advanced"     // продвинутый
)

// Role описывает роль пользователя в системе.
type Role string

const (
	RoleUser  Role = "user"
	RoleCoach Role = "coach"
	RoleAdmin Role = "admin"
)

// User представляет доменную модель пользователя фитнес‑приложения.
//
// Важно: эта модель описывает бизнес‑сущность и не зависит от деталей транспорта (HTTP, gRPC)
// и конкретного представления в БД.
type User struct {
	ID           uuid.UUID // Уникальный идентификатор пользователя
	Email        string    // Email (уникальный логин)
	PasswordHash string    // Хэш пароля
	Username     string    // Никнейм (уникальный)

	FirstName string     // Имя
	LastName  string     // Фамилия
	BirthDate *time.Time // Дата рождения (опционально)
	Gender    string     // Пол (опционально, свободная строка или отдельный enum позже)
	AvatarURL string     // URL аватара (опционально)
	Role      Role       // Роль (user/coach/admin и т.п.)

	TrainingLevel   TrainingLevel // Уровень подготовки
	IsEmailVerified bool          // Подтверждён ли email пользователя

	CreatedAt time.Time  // Время создания
	UpdatedAt time.Time  // Время последнего обновления
	DeletedAt *time.Time // Для мягкого удаления (nil, если активен)
}

// NewUser — фабрика для создания нового пользователя на доменном уровне.
// Предполагается, что валидация/нормализация входных данных и хеширование пароля
// выполняются на уровне usecase‑слоя до вызова этой функции.
func NewUser(
	email string,
	passwordHash string,
	username string,
) *User {
	now := time.Now().UTC()
	return &User{
		ID:            uuid.New(),
		Email:         email,
		PasswordHash:  passwordHash,
		Username:      username,
		Role:          RoleUser,
		TrainingLevel: TrainingLevelBeginner,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// IsDeleted возвращает true, если пользователь мягко удалён.
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// MarkDeleted помечает пользователя как удалённого и обновляет время обновления.
func (u *User) MarkDeleted(at time.Time) {
	u.DeletedAt = &at
	u.UpdatedAt = at
}

// Touch обновляет время последнего изменения сущности.
func (u *User) Touch(at time.Time) {
	u.UpdatedAt = at
}

// EmailVerification представляет доменную модель кода подтверждения email.
type EmailVerification struct {
	ID          int64     // Идентификатор записи (соответствует BIGSERIAL в БД)
	UserID      uuid.UUID // Пользователь, для которого создан код
	CodeHash    string    // Хэш одноразового кода подтверждения
	ExpiresAt   time.Time // Время истечения кода
	Attempts    int       // Количество использованных попыток
	MaxAttempts int       // Максимально допустимое количество попыток
	CreatedAt   time.Time // Время создания записи
}
