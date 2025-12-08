//go:build integration
// +build integration

package user_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	authhandler "workout-app/internal/handler/auth"
	userhandler "workout-app/internal/handler/user"
	testcfg "workout-app/tests/integration/config"
)

// registerAndLoginUser регистрирует пользователя, подтверждает email и выполняет логин.
// Возвращает userID и accessToken.
func registerAndLoginUser(t *testing.T, router *gin.Engine, email, password, username string) (string, string) {
	t.Helper()

	// Регистрация
	registerBody := `{"email":"` + email + `","password":"` + password + `","username":"` + username + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	require.Equal(t, email, regResp.Email)
	require.Equal(t, username, regResp.Username)
	userID := regResp.UserID

	// Подтверждение email
	testcfg.VerifyUserEmailForTests(t, email)

	// Логин
	loginBody := `{"email":"` + email + `","password":"` + password + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	accessToken := loginResp.Tokens.AccessToken
	require.NotEmpty(t, accessToken)

	return userID, accessToken
}

// deleteUserAccount удаляет аккаунт пользователя с подтверждением пароля.
func deleteUserAccount(t *testing.T, router *gin.Engine, accessToken, password string) {
	t.Helper()

	deleteBody := `{"password":"` + password + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", strings.NewReader(deleteBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
}

// TestUser_Profile_Flow проверяет сценарий:
// register -> /users/me -> update -> /users/me -> delete -> /users/me (404) ->
// login (403 account_deleted) -> restore -> /users/me (200).
func TestUser_Profile_Flow(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация, подтверждение email и логин
	userID, access := registerAndLoginUser(t, router, "uflow@example.com", "Password123!", "uflow")

	// 2. GET /users/me
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
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
	deleteUserAccount(t, router, access, "Password123!")

	// 5. GET /users/me после удаления -> 404
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// 6. Попытка логина после удаления -> 403 account_deleted
	loginBody := `{"email":"uflow@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "account_deleted", errorResp["error"].(map[string]interface{})["code"])

	// 7. Восстановление аккаунта через POST /api/v1/users/restore -> возвращает профиль и токены
	restoreBody := `{"email":"uflow@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/restore", strings.NewReader(restoreBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var restoredResp userhandler.RestoreAccountResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restoredResp))
	require.Equal(t, userID, restoredResp.UserID)
	require.Equal(t, "uflow@example.com", restoredResp.Email)
	require.Equal(t, "uflownew", restoredResp.Username)
	require.NotEmpty(t, restoredResp.Tokens.AccessToken)
	require.NotEmpty(t, restoredResp.Tokens.RefreshToken)

	// 8. Проверка что профиль доступен через GET /users/me с токеном из восстановления
	newAccess := restoredResp.Tokens.AccessToken
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+newAccess)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var restoredProfile userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restoredProfile))
	require.Equal(t, "uflownew", restoredProfile.Username)
	require.Equal(t, "intermediate", restoredProfile.TrainingLevel)
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
// попытка регистрации (409) -> восстановление (возвращает токены) -> проверка профиля.
func TestUser_Restore_Account(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация, подтверждение email и удаление аккаунта
	userID, access := registerAndLoginUser(t, router, "restore@example.com", "Password123!", "restoreuser")
	deleteUserAccount(t, router, access, "Password123!")

	// 5. Попытка логина -> должна вернуть 403 с account_deleted
	loginBody := `{"email":"restore@example.com","password":"Password123!"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "account_deleted", errorResp["error"].(map[string]interface{})["code"])

	// 6. Попытка регистрации с тем же email -> должна вернуть 409 с account_deleted
	registerBody := `{"email":"restore@example.com","password":"NewPassword123!","username":"newuser"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "account_deleted", errorResp["error"].(map[string]interface{})["code"])

	// 7. Восстановление аккаунта через POST /api/v1/users/restore -> 200 с токенами
	restoreBody := `{"email":"restore@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/restore", strings.NewReader(restoreBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var restoredResp userhandler.RestoreAccountResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restoredResp))
	require.Equal(t, userID, restoredResp.UserID)
	require.Equal(t, "restore@example.com", restoredResp.Email)
	require.Equal(t, "restoreuser", restoredResp.Username)
	require.NotEmpty(t, restoredResp.Tokens.AccessToken)
	require.NotEmpty(t, restoredResp.Tokens.RefreshToken)

	// 8. Проверка что профиль доступен через GET /users/me с токеном из восстановления
	newAccess := restoredResp.Tokens.AccessToken
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

// TestUser_Restore_Account_InvalidCredentials проверяет восстановление аккаунта с неверным паролем.
func TestUser_Restore_Account_InvalidCredentials(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация, подтверждение email и удаление аккаунта
	_, access := registerAndLoginUser(t, router, "restoreinvalid@example.com", "Password123!", "restoreinvalid")
	deleteUserAccount(t, router, access, "Password123!")

	// 2. Попытка восстановления с неверным паролем -> 401 invalid_credentials
	restoreBody := `{"email":"restoreinvalid@example.com","password":"WrongPassword123!"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/restore", strings.NewReader(restoreBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())

	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "invalid_credentials", errorResp["error"].(map[string]interface{})["code"])
}

// TestUser_Restore_Account_NotDeleted проверяет попытку восстановления не удалённого аккаунта.
func TestUser_Restore_Account_NotDeleted(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация и подтверждение email (аккаунт не удалён)
	registerAndLoginUser(t, router, "restorenotdeleted@example.com", "Password123!", "restorenotdeleted")

	// 2. Попытка восстановления не удалённого аккаунта -> 400 account_not_deleted
	restoreBody := `{"email":"restorenotdeleted@example.com","password":"Password123!"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/restore", strings.NewReader(restoreBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "account_not_deleted", errorResp["error"].(map[string]interface{})["code"])
}

// TestUser_Restore_Account_EmailNotVerified проверяет восстановление аккаунта с неподтверждённым email.
func TestUser_Restore_Account_EmailNotVerified(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация без подтверждения email
	registerBody := `{"email":"restoreunverified@example.com","password":"Password123!","username":"restoreunverified"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))

	// 2. Удаление аккаунта (без подтверждения email)
	testcfg.SoftDeleteUserForTests(t, regResp.Email)

	// 3. Попытка восстановления аккаунта с неподтверждённым email -> 403 email_not_verified
	restoreBody := `{"email":"restoreunverified@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/restore", strings.NewReader(restoreBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "email_not_verified", errorResp["error"].(map[string]interface{})["code"])
}
