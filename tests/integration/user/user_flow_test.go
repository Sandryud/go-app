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

	// 4. DELETE /users/me (с подтверждением пароля)
	deleteBody := `{"password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", strings.NewReader(deleteBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	// 5. GET /users/me после удаления -> 404
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// 6. Попытка логина после удаления -> 403 account_deleted
	loginBody = `{"email":"uflow@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "account_deleted", errorResp["error"].(map[string]interface{})["code"])

	// 7. Восстановление аккаунта через POST /api/v1/users/restore
	restoreBody := `{"email":"uflow@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/restore", strings.NewReader(restoreBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var restoredProfile userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restoredProfile))
	require.Equal(t, "uflownew", restoredProfile.Username)
	require.Equal(t, "intermediate", restoredProfile.TrainingLevel)

	// 8. Повторный логин после восстановления -> должен быть успешным
	loginBody = `{"email":"uflow@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginRespAfterRestore authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginRespAfterRestore))
	require.NotEmpty(t, loginRespAfterRestore.Tokens.AccessToken)
	require.NotEmpty(t, loginRespAfterRestore.Tokens.RefreshToken)
}

// TestUser_GetByID проверяет endpoint GET /api/v1/users/:id:
// успешное получение публичного профиля другим пользователем, 404 для несуществующего, 400 для невалидного UUID, 401 без авторизации.
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

	// 5. GET /users/:id - несуществующий пользователь -> 404
	nonExistentID := "00000000-0000-0000-0000-000000000000"
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+nonExistentID, nil)
	req.Header.Set("Authorization", "Bearer "+access1)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// 6. GET /users/:id - невалидный UUID -> 400
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+access1)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// 7. GET /users/:id - без авторизации -> 401
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user1ID, nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}

// TestUser_Restore_Account проверяет полный сценарий восстановления аккаунта:
// регистрация -> подтверждение email -> логин -> удаление -> попытка логина (403) ->
// попытка регистрации (409) -> восстановление -> повторный логин -> проверка профиля.
func TestUser_Restore_Account(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация пользователя
	registerBody := `{"email":"restore@example.com","password":"Password123!","username":"restoreuser"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	require.Equal(t, "restore@example.com", regResp.Email)
	require.Equal(t, "restoreuser", regResp.Username)
	userID := regResp.UserID

	// 2. Подтверждение email
	testcfg.VerifyUserEmailForTests(t, regResp.Email)

	// 3. Логин и получение токена
	loginBody := `{"email":"restore@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	access := loginResp.Tokens.AccessToken
	require.NotEmpty(t, access)

	// 4. Удаление аккаунта
	deleteBody := `{"password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", strings.NewReader(deleteBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	// 5. Попытка логина -> должна вернуть 403 с account_deleted
	loginBody = `{"email":"restore@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "account_deleted", errorResp["error"].(map[string]interface{})["code"])

	// 6. Попытка регистрации с тем же email -> должна вернуть 409 с account_deleted
	registerBody = `{"email":"restore@example.com","password":"NewPassword123!","username":"newuser"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "account_deleted", errorResp["error"].(map[string]interface{})["code"])

	// 7. Восстановление аккаунта через POST /api/v1/users/restore -> 200
	restoreBody := `{"email":"restore@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/restore", strings.NewReader(restoreBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var restoredProfile userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restoredProfile))
	require.Equal(t, userID, restoredProfile.ID)
	require.Equal(t, "restore@example.com", restoredProfile.Email)
	require.Equal(t, "restoreuser", restoredProfile.Username)

	// 8. Повторный логин после восстановления -> должен быть успешным
	loginBody = `{"email":"restore@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginRespAfterRestore authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginRespAfterRestore))
	require.Equal(t, userID, loginRespAfterRestore.UserID)
	require.NotEmpty(t, loginRespAfterRestore.Tokens.AccessToken)
	require.NotEmpty(t, loginRespAfterRestore.Tokens.RefreshToken)

	// 9. Проверка что профиль доступен через GET /users/me
	newAccess := loginRespAfterRestore.Tokens.AccessToken
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+newAccess)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var profile userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &profile))
	require.Equal(t, userID, profile.ID)
	require.Equal(t, "restore@example.com", profile.Email)
	require.Equal(t, "restoreuser", profile.Username)
}
