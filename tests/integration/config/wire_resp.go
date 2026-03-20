//go:build integration
// +build integration

package config

import "time"

// Ниже — фактический формат JSON ответов API: ginjson.API сериализует ключи в camelCase
// (см. internal/server.NewServer и pkg/jsoncodec).

// RegisterResp — тело ответа POST /auth/register.
type RegisterResp struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// TokenPairResp — пара токенов в ответах login / refresh / restore.
type TokenPairResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// LoginResp — ответ login, refresh и восстановления аккаунта (одинаковая форма).
type LoginResp struct {
	UserID   string        `json:"userId"`
	Email    string        `json:"email"`
	Username string        `json:"username"`
	Tokens   TokenPairResp `json:"tokens"`
}

// ProfileResp — GET/PUT /users/me.
type ProfileResp struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	FirstName     string     `json:"firstName,omitempty"`
	LastName      string     `json:"lastName,omitempty"`
	BirthDate     *time.Time `json:"birthDate,omitempty"`
	Gender        string     `json:"gender,omitempty"`
	AvatarURL     string     `json:"avatarUrl,omitempty"`
	Role          string     `json:"role,omitempty"`
	TrainingLevel string     `json:"trainingLevel,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// PublicProfileResp — GET /users/:id.
type PublicProfileResp struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	FirstName     string     `json:"firstName,omitempty"`
	LastName      string     `json:"lastName,omitempty"`
	BirthDate     *time.Time `json:"birthDate,omitempty"`
	Gender        string     `json:"gender,omitempty"`
	AvatarURL     string     `json:"avatarUrl,omitempty"`
	Role          string     `json:"role,omitempty"`
	TrainingLevel string     `json:"trainingLevel,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// ChangeEmailResp — ответ POST /users/me/change-email.
type ChangeEmailResp struct {
	Message string `json:"message"`
}
