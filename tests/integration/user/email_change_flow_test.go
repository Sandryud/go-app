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

// TestUser_EmailChange_Flow проверяет полный флоу изменения email:
// register -> login -> change-email -> verify-email-change -> проверка нового email.
func TestUser_EmailChange_Flow(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация
	email := "emailchange1@example.com"
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"emailchange1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	userID := regResp.UserID

	// Форсируем подтверждение email для получения токенов через логин
	testcfg.VerifyUserEmailForTests(t, email)

	// 2. Логин
	loginBody := `{"email":"` + email + `","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	access := loginResp.Tokens.AccessToken

	// 3. Проверяем текущий email
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var profileBefore userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &profileBefore))
	require.Equal(t, email, profileBefore.Email)

	// 4. Запрос на изменение email
	newEmail := "newemail1@example.com"
	changeEmailBody := `{"new_email":"` + newEmail + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-email", strings.NewReader(changeEmailBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var changeEmailResp userhandler.ChangeEmailResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &changeEmailResp))
	require.Contains(t, changeEmailResp.Message, "Код подтверждения отправлен")

	// 5. Форсируем изменение email для теста (в реальном сценарии здесь был бы код из письма)
	testcfg.ForceEmailChangeForTests(t, userID, newEmail)

	// 6. Проверяем, что email изменился
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var profileAfter userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &profileAfter))
	require.Equal(t, newEmail, profileAfter.Email)
	require.Equal(t, profileBefore.Username, profileAfter.Username)
}

// TestUser_EmailChange_SameEmail проверяет ошибку при попытке изменить email на тот же самый.
func TestUser_EmailChange_SameEmail(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "sameemail@example.com"
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"sameemail"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email)

	loginBody := `{"email":"` + email + `","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	access := loginResp.Tokens.AccessToken

	// Попытка изменить email на тот же самый
	changeEmailBody := `{"new_email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-email", strings.NewReader(changeEmailBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// TestUser_EmailChange_EmailAlreadyExists проверяет ошибку при попытке изменить email на уже занятый.
func TestUser_EmailChange_EmailAlreadyExists(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// Регистрация первого пользователя
	email1 := "user1@example.com"
	registerBody1 := `{"email":"` + email1 + `","password":"Password123!","username":"user1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody1))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email1)

	// Регистрация второго пользователя
	email2 := "user2@example.com"
	registerBody2 := `{"email":"` + email2 + `","password":"Password123!","username":"user2"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody2))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email2)

	// Логин второго пользователя
	loginBody2 := `{"email":"` + email2 + `","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody2))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp2 authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp2))
	access2 := loginResp2.Tokens.AccessToken

	// Попытка второго пользователя изменить email на email первого пользователя
	changeEmailBody := `{"new_email":"` + email1 + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-email", strings.NewReader(changeEmailBody))
	req.Header.Set("Authorization", "Bearer "+access2)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

// TestUser_EmailChange_VerifyEmailChange_InvalidCode проверяет ошибку при неверном коде подтверждения.
func TestUser_EmailChange_VerifyEmailChange_InvalidCode(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "invalidcode@example.com"
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"invalidcode"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email)

	loginBody := `{"email":"` + email + `","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	access := loginResp.Tokens.AccessToken

	// Запрос на изменение email
	newEmail := "newinvalid@example.com"
	changeEmailBody := `{"new_email":"` + newEmail + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-email", strings.NewReader(changeEmailBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Попытка подтвердить с неверным кодом
	verifyBody := `{"code":"000000"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/verify-email-change", strings.NewReader(verifyBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// TestUser_EmailChange_VerifyEmailChange_CodeNotFound проверяет ошибку при отсутствии активного кода.
func TestUser_EmailChange_VerifyEmailChange_CodeNotFound(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "nocode@example.com"
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"nocode"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email)

	loginBody := `{"email":"` + email + `","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	access := loginResp.Tokens.AccessToken

	// Попытка подтвердить изменение email без предварительного запроса
	verifyBody := `{"code":"123456"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/verify-email-change", strings.NewReader(verifyBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
