//go:build integration
// +build integration

package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authhandler "workout-app/internal/handler/auth"
	"workout-app/pkg/password"
	testcfg "workout-app/tests/integration/config"
)

// TestAuth_PasswordReset_FullFlow проверяет happy-path сброса пароля:
// register -> verify email -> login -> forgot-password -> reset-password -> login с новым паролем.
func TestAuth_PasswordReset_FullFlow(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "pwreset1@example.com"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 1. Регистрация
	registerBody := `{"email":"` + email + `","password":"` + oldPassword + `","username":"pwreset1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	require.Equal(t, email, regResp.Email)

	// 2. Подтверждение email
	testcfg.VerifyUserEmailForTests(t, email)

	// 3. Логин со старым паролем
	loginBody := `{"email":"` + email + `","password":"` + oldPassword + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 4. Запрос кода сброса пароля
	forgotBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(forgotBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var forgotResp authhandler.ForgotPasswordResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &forgotResp))
	require.Contains(t, forgotResp.Message, "password reset code has been sent")

	// 5. Создаем код сброса пароля для теста (в реальности код приходит на email)
	testCode := "123456"
	codeHash, err := password.Hash(testCode)
	require.NoError(t, err)
	testcfg.CreatePasswordResetCodeForTests(t, email, codeHash)

	// 6. Сброс пароля
	resetBody := `{"email":"` + email + `","code":"` + testCode + `","new_password":"` + newPassword + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(resetBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resetResp authhandler.ResetPasswordResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resetResp))
	require.Contains(t, resetResp.Message, "successfully reset")

	// 7. Проверка: старый пароль больше не работает
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())

	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "invalid_credentials", errorResp["error"].(map[string]interface{})["code"])

	// 8. Проверка: новый пароль работает
	newLoginBody := `{"email":"` + email + `","password":"` + newPassword + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(newLoginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var newLoginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newLoginResp))
	require.Equal(t, regResp.UserID, newLoginResp.UserID)
	require.NotEmpty(t, newLoginResp.Tokens.AccessToken)
}

// TestAuth_PasswordReset_UnverifiedEmail проверяет, что нельзя запросить сброс пароля
// для неподтвержденного email.
func TestAuth_PasswordReset_UnverifiedEmail(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "unverified@example.com"

	// 1. Регистрация (без подтверждения email)
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"unverified"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// 2. Попытка запросить сброс пароля для неподтвержденного email
	forgotBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(forgotBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "email_not_verified", errorResp["error"].(map[string]interface{})["code"])
}

// TestAuth_PasswordReset_ExpiredCode проверяет, что истекший код сброса пароля не работает.
func TestAuth_PasswordReset_ExpiredCode(t *testing.T) {
	// Делаем TTL очень коротким для теста.
	os.Setenv("EMAIL_VERIFICATION_TTL", "1s")
	t.Cleanup(func() {
		os.Unsetenv("EMAIL_VERIFICATION_TTL")
	})

	router := testcfg.NewTestRouter(t)

	email := "expiredcode@example.com"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 1. Регистрация и подтверждение email
	registerBody := `{"email":"` + email + `","password":"` + oldPassword + `","username":"expiredcode"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email)

	// 2. Запрос кода сброса пароля
	forgotBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(forgotBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 3. Создаем код с истекшим сроком действия
	testCode := "123456"
	codeHash, err := password.Hash(testCode)
	require.NoError(t, err)

	// Создаем код с нормальным сроком, затем обновим его на истекший
	testcfg.CreatePasswordResetCodeForTests(t, email, codeHash)

	// Обновляем expires_at на прошедшее время через прямой SQL
	db := testcfg.GetTestDB()
	require.NotNil(t, db, "test database should be initialized")

	var userID string
	if err := db.Raw(`SELECT id::text FROM users WHERE email = $1`, email).Scan(&userID).Error; err != nil {
		t.Fatalf("failed to get user ID: %v", err)
	}
	if err := db.Exec(`UPDATE password_resets SET expires_at = NOW() - INTERVAL '1 minute' WHERE user_id = $1`, userID).Error; err != nil {
		t.Fatalf("failed to expire password reset code: %v", err)
	}

	// Ждем немного, чтобы убедиться, что код истек
	time.Sleep(2 * time.Second)

	// 4. Попытка сброса пароля с истекшим кодом
	resetBody := `{"email":"` + email + `","code":"` + testCode + `","new_password":"` + newPassword + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(resetBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "password_reset_code_not_found", errorResp["error"].(map[string]interface{})["code"])
}

// TestAuth_PasswordReset_MaxAttempts проверяет, что превышение лимита попыток блокирует сброс пароля.
func TestAuth_PasswordReset_MaxAttempts(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "maxattempts@example.com"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 1. Регистрация и подтверждение email
	registerBody := `{"email":"` + email + `","password":"` + oldPassword + `","username":"maxattempts"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email)

	// 2. Запрос кода сброса пароля
	forgotBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(forgotBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 3. Создаем код сброса пароля для теста
	testCode := "123456"
	codeHash, err := password.Hash(testCode)
	require.NoError(t, err)
	testcfg.CreatePasswordResetCodeForTests(t, email, codeHash)

	// 4. Многократные неверные попытки сброса пароля до превышения лимита
	for i := 0; i < 5; i++ { // 5 == default max attempts
		resetBody := `{"email":"` + email + `","code":"000000","new_password":"` + newPassword + `"}`
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(resetBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if i < 4 {
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String(), "ожидалась ошибка неверного кода до превышения лимита попыток")
			var errorResp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
			require.Equal(t, "password_reset_code_invalid", errorResp["error"].(map[string]interface{})["code"])
		} else {
			// На последней попытке ожидаем ошибку превышения попыток
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var errorResp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
			require.Equal(t, "password_reset_attempts_exceeded", errorResp["error"].(map[string]interface{})["code"])
		}
	}
}

// TestAuth_PasswordReset_CodeUsedTwice проверяет, что код сброса пароля можно использовать только один раз.
func TestAuth_PasswordReset_CodeUsedTwice(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "usedtwice@example.com"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 1. Регистрация и подтверждение email
	registerBody := `{"email":"` + email + `","password":"` + oldPassword + `","username":"usedtwice"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	testcfg.VerifyUserEmailForTests(t, email)

	// 2. Запрос кода сброса пароля
	forgotBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(forgotBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 3. Создаем код сброса пароля для теста
	testCode := "123456"
	codeHash, err := password.Hash(testCode)
	require.NoError(t, err)
	testcfg.CreatePasswordResetCodeForTests(t, email, codeHash)

	// 4. Первый сброс пароля (успешный)
	resetBody := `{"email":"` + email + `","code":"` + testCode + `","new_password":"` + newPassword + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(resetBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 5. Попытка использовать тот же код повторно (должна вернуть ошибку)
	// Код уже удален после успешного сброса, поэтому попытка использовать его снова
	// должна вернуть ошибку "code not found"

	// Попытка использовать код повторно (код уже удален после первого использования)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(resetBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResp))
	require.Equal(t, "password_reset_code_not_found", errorResp["error"].(map[string]interface{})["code"])
}
