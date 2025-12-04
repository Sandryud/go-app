package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
	"workout-app/internal/config"
)

// Claims описывает JWT-пейлоад, который мы используем для access и refresh токенов.
type Claims struct {
	UserID        string `json:"sub"`
	Email         string `json:"email,omitempty"`
	Username      string `json:"username,omitempty"`
	Role          string `json:"role,omitempty"`
	TrainingLevel string `json:"training_level,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	jwt.RegisteredClaims
}

// Service инкапсулирует операции по генерации и валидации JWT-токенов.
type Service interface {
	GenerateAccessToken(user *domain.User) (string, error)
	GenerateRefreshToken(user *domain.User) (string, string, error) // token, jti
	ParseAccessToken(tokenString string) (*Claims, error)
	ParseRefreshToken(tokenString string) (*Claims, error)
}

type service struct {
	cfg *config.JWTConfig
}

// NewService создаёт JWT-сервис на основе конфигурации.
func NewService(cfg *config.JWTConfig) Service {
	return &service{cfg: cfg}
}

// GenerateAccessToken генерирует короткоживущий access-токен для пользователя.
func (s *service) GenerateAccessToken(user *domain.User) (string, error) {
	now := time.Now().UTC()
	claims := &Claims{
		UserID:        user.ID.String(),
		Email:         user.Email,
		Username:      user.Username,
		Role:          string(user.Role),
		TrainingLevel: string(user.TrainingLevel),
		EmailVerified: user.IsEmailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.AccessSecret))
}

// GenerateRefreshToken генерирует долгоживущий refresh-токен для пользователя и возвращает его jti.
func (s *service) GenerateRefreshToken(user *domain.User) (string, string, error) {
	now := time.Now().UTC()
	jti := uuid.New().String()

	claims := &Claims{
		UserID:        user.ID.String(),
		Email:         user.Email,
		Username:      user.Username,
		Role:          string(user.Role),
		EmailVerified: user.IsEmailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   user.ID.String(),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.RefreshTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.RefreshSecret))
	if err != nil {
		return "", "", err
	}
	return signed, jti, nil
}

// ParseAccessToken парсит и валидирует access-токен.
func (s *service) ParseAccessToken(tokenString string) (*Claims, error) {
	return s.parseToken(tokenString, []byte(s.cfg.AccessSecret))
}

// ParseRefreshToken парсит и валидирует refresh-токен.
func (s *service) ParseRefreshToken(tokenString string) (*Claims, error) {
	return s.parseToken(tokenString, []byte(s.cfg.RefreshSecret))
}

// parseToken — общая логика парсинга JWT.
func (s *service) parseToken(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Дополнительная защита: убеждаемся, что метод подписи ожидаемый
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	// Дополнительная проверка issuer при необходимости
	if claims.Issuer != "" && s.cfg.Issuer != "" && claims.Issuer != s.cfg.Issuer {
		return nil, jwt.ErrTokenInvalidIssuer
	}

	return claims, nil
}


