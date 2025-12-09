package auth

import "workout-app/internal/handler/response"

// RegisterRequest описывает тело запроса регистрации пользователя.
// Контракт намеренно минимальный: только данные, необходимые для аутентификации.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	// Username должен состоять только из букв и цифр (без пробелов и спецсимволов).
	Username string `json:"username" binding:"required,alphanum,min=3,max=32"`
}

// RegisterResponse описывает ответ при успешной регистрации (отправке кода подтверждения).
type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// LoginRequest описывает тело запроса логина.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// VerifyEmailRequest описывает тело запроса подтверждения email кодом.
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

// ResendVerificationRequest описывает тело запроса повторной отправки кода.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResendVerificationResponse описывает ответ на повторную отправку кода.
type ResendVerificationResponse struct {
	Message string `json:"message"`
}

// LoginResponse — ответ при успешной аутентификации/регистрации.
// Содержит пару токенов и базовую идентифицирующую информацию о пользователе.
type LoginResponse struct {
	UserID   string             `json:"user_id"`
	Email    string             `json:"email"`
	Username string             `json:"username"`
	Tokens   response.TokenPair `json:"tokens"`
}

// RefreshRequest описывает тело запроса обновления токенов.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RestoreAccountRequest описывает тело запроса для восстановления аккаунта.
type RestoreAccountRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RestoreAccountResponse описывает ответ при успешном восстановлении аккаунта.
type RestoreAccountResponse struct {
	UserID   string             `json:"user_id"`
	Email    string             `json:"email"`
	Username string             `json:"username"`
	Tokens   response.TokenPair `json:"tokens"`
}

// ForgotPasswordRequest описывает тело запроса для запроса кода сброса пароля.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPasswordResponse описывает ответ на запрос кода сброса пароля.
type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

// ResetPasswordRequest описывает тело запроса для сброса пароля.
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPasswordResponse описывает ответ при успешном сбросе пароля.
type ResetPasswordResponse struct {
	Message string `json:"message"`
}
