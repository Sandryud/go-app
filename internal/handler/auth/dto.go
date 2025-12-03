package auth

// RegisterRequest описывает тело запроса регистрации пользователя.
// Контракт намеренно минимальный: только данные, необходимые для аутентификации.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Username string `json:"username" binding:"required,min=3,max=32"`
}

// LoginRequest описывает тело запроса логина.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TokenPair описывает пару access/refresh токенов.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// LoginResponse — ответ при успешной аутентификации/регистрации.
// Содержит пару токенов и базовую идентифицирующую информацию о пользователе.
type LoginResponse struct {
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Username string    `json:"username"`
	Tokens   TokenPair `json:"tokens"`
}

// RefreshRequest описывает тело запроса обновления токенов.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}


