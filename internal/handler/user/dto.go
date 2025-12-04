package user

import "time"

// ProfileResponse описывает профиль текущего пользователя.
// Этот контракт используется в защищённых эндпоинтах (/api/v1/users/me и т.п.).
type ProfileResponse struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	FirstName     string     `json:"first_name,omitempty"`
	LastName      string     `json:"last_name,omitempty"`
	BirthDate     *time.Time `json:"birth_date,omitempty"`
	Gender        string     `json:"gender,omitempty"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Role          string     `json:"role,omitempty"`
	TrainingLevel string     `json:"training_level,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ProfileUpdateRequest описывает тело запроса для отдельного эндпоинта
// обновления профильной информации пользователя.
type ProfileUpdateRequest struct {
	// Username при обновлении также ограничен только буквами и цифрами.
	Username      *string    `json:"username,omitempty" binding:"omitempty,alphanum,min=3,max=32"`
	FirstName     *string    `json:"first_name,omitempty"`
	LastName      *string    `json:"last_name,omitempty"`
	BirthDate     *time.Time `json:"birth_date,omitempty"`
	Gender        *string    `json:"gender,omitempty"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	TrainingLevel *string    `json:"training_level,omitempty"`
}

// PublicProfileResponse описывает публичный профиль пользователя.
// Используется в эндпоинтах, где не требуется отображать приватную информацию (например, email).
type PublicProfileResponse struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	FirstName     string     `json:"first_name,omitempty"`
	LastName      string     `json:"last_name,omitempty"`
	BirthDate     *time.Time `json:"birth_date,omitempty"`
	Gender        string     `json:"gender,omitempty"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Role          string     `json:"role,omitempty"`
	TrainingLevel string     `json:"training_level,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ChangeEmailRequest описывает тело запроса для изменения email.
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

// ChangeEmailResponse описывает ответ на запрос изменения email.
type ChangeEmailResponse struct {
	Message string `json:"message"`
}

// VerifyEmailChangeRequest описывает тело запроса для подтверждения изменения email.
type VerifyEmailChangeRequest struct {
	Code string `json:"code" binding:"required,len=6,numeric"`
}
