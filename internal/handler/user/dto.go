package user

import "time"

// ProfileResponse описывает профиль текущего пользователя.
// Этот контракт используется в защищённых эндпоинтах (/api/v1/users/me и т.п.).
type ProfileResponse struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Username  string     `json:"username"`
	FirstName string     `json:"first_name,omitempty"`
	LastName  string     `json:"last_name,omitempty"`
	BirthDate *time.Time `json:"birth_date,omitempty"`
	Gender    string     `json:"gender,omitempty"`
	AvatarURL string     `json:"avatar_url,omitempty"`
	Role      string     `json:"role,omitempty"`
	TrainingLevel string `json:"training_level,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ProfileUpdateRequest описывает тело запроса для отдельного эндпоинта
// обновления профильной информации пользователя.
type ProfileUpdateRequest struct {
	Username    *string   `json:"username,omitempty"`
	FirstName   *string   `json:"first_name,omitempty"`
	LastName    *string   `json:"last_name,omitempty"`
	BirthDate   *time.Time `json:"birth_date,omitempty"`
	Gender      *string   `json:"gender,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	TrainingLevel *string `json:"training_level,omitempty"`
}


