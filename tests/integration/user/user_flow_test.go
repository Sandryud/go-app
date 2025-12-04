//go:build integration
// +build integration

package user_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	authhandler "workout-app/internal/handler/auth"
	userhandler "workout-app/internal/handler/user"
	testcfg "workout-app/tests/integration/config"
)

// TestUser_Profile_Flow проверяет сценарий:
// register -> /users/me -> update -> /users/me -> delete -> /users/me (404).
func TestUser_Profile_Flow(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация
	registerBody := `{"email":"uflow@example.com","password":"Password123!","username":"uflow"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	require.Equal(t, "uflow@example.com", regResp.Email)
	require.Equal(t, "uflow", regResp.Username)
	require.NotEmpty(t, regResp.UserID)

	// Форсируем подтверждение email в БД для получения токенов через логин.
	testcfg.VerifyUserEmailForTests(t, regResp.Email)

	// Выполняем логин, чтобы получить access-токен.
	loginBody := `{"email":"uflow@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	access := loginResp.Tokens.AccessToken

	// 2. GET /users/me
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var profile userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &profile))
	require.Equal(t, "uflow", profile.Username)

	// 3. PUT /users/me (обновление профиля)
	updateBody := `{"username":"uflownew","first_name":"Test","training_level":"intermediate"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/me", strings.NewReader(updateBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var updated userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	require.Equal(t, "uflownew", updated.Username)
	require.Equal(t, "intermediate", updated.TrainingLevel)

	// 4. DELETE /users/me
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	// 5. GET /users/me после удаления -> 404
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

// TestUser_GetByID проверяет endpoint GET /api/v1/users/:id:
// успешное получение, проверка публичного профиля (без email), 404 для несуществующего, 400 для невалидного UUID.
func TestUser_GetByID(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация первого пользователя
	registerBody1 := `{"email":"testuser@example.com","password":"Password123!","username":"testuser"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody1))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp1 authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp1))
	user1ID := regResp1.UserID

	// Форсируем подтверждение email первого пользователя и логинимся.
	testcfg.VerifyUserEmailForTests(t, regResp1.Email)
	loginBody1 := `{"email":"testuser@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody1))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp1 authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp1))
	access1 := loginResp1.Tokens.AccessToken

	// 2. Обновление профиля первого пользователя для проверки данных
	updateBody := `{"first_name":"Иван","last_name":"Иванов","training_level":"intermediate"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/me", strings.NewReader(updateBody))
	req.Header.Set("Authorization", "Bearer "+access1)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 3. Регистрация второго пользователя для тестирования доступа
	registerBody2 := `{"email":"testuser2@example.com","password":"Password123!","username":"testuser2"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody2))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp2 authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp2))

	// Форсируем подтверждение email второго пользователя и логинимся.
	testcfg.VerifyUserEmailForTests(t, regResp2.Email)
	loginBody2 := `{"email":"testuser2@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody2))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp2 authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp2))
	access2 := loginResp2.Tokens.AccessToken

	// 4. GET /users/:id - успешное получение публичного профиля (второй пользователь получает профиль первого)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user1ID, nil)
	req.Header.Set("Authorization", "Bearer "+access2)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var publicProfile userhandler.PublicProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &publicProfile))
	require.Equal(t, user1ID, publicProfile.ID)
	require.Equal(t, "testuser", publicProfile.Username)
	require.Equal(t, "Иван", publicProfile.FirstName)
	require.Equal(t, "Иванов", publicProfile.LastName)
	require.Equal(t, "intermediate", publicProfile.TrainingLevel)
	// Проверяем, что email отсутствует в публичном профиле
	// (поле Email не должно быть в структуре PublicProfileResponse, но проверяем через JSON)
	var profileMap map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &profileMap))
	_, emailExists := profileMap["email"]
	require.False(t, emailExists, "email не должен присутствовать в публичном профиле")

	// 5. GET /users/:id - получение собственного профиля
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user1ID, nil)
	req.Header.Set("Authorization", "Bearer "+access1)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var ownProfile userhandler.PublicProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ownProfile))
	require.Equal(t, user1ID, ownProfile.ID)

	// 6. GET /users/:id - несуществующий пользователь -> 404
	nonExistentID := "00000000-0000-0000-0000-000000000000"
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+nonExistentID, nil)
	req.Header.Set("Authorization", "Bearer "+access1)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// 7. GET /users/:id - невалидный UUID -> 400
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+access1)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// 8. GET /users/:id - без авторизации -> 401
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user1ID, nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}


