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
	testcfg "workout-app/tests/integration/config"
)

// TestAuth_Register_Resend_Verify_Login проверяет сценарий:
// register -> resend-verification -> (форсированное подтверждение) -> login.
func TestAuth_Register_Resend_Verify_Login(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "iresend1@example.com"

	// 1. Регистрация
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"iresend1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	require.Equal(t, email, regResp.Email)

	// 2. Повторная отправка кода подтверждения
	resendBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", strings.NewReader(resendBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 3. Форсируем подтверждение email для успешного логина.
	testcfg.VerifyUserEmailForTests(t, email)

	// 4. Вход в систему
	loginBody := `{"email":"` + email + `","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	require.Equal(t, regResp.UserID, loginResp.UserID)
	require.NotEmpty(t, loginResp.Tokens.AccessToken)
}

// TestAuth_Verify_ExpiredCode_Resend_Verify проверяет:
// register -> истечь TTL -> verify (expired) -> resend-verification.
func TestAuth_Verify_ExpiredCode_Resend_Verify(t *testing.T) {
	// Делаем TTL очень коротким для теста.
	os.Setenv("EMAIL_VERIFICATION_TTL", "1s")
	t.Cleanup(func() {
		os.Unsetenv("EMAIL_VERIFICATION_TTL")
	})

	router := testcfg.NewTestRouter(t)

	email := "expired1@example.com"

	// 1. Регистрация
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"expired1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// Ждём истечения TTL
	time.Sleep(2 * time.Second)

	// 2. Попытка подтверждения с любым кодом должна дать expired/not found.
	verifyBody := `{"email":"` + email + `","code":"000000"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", strings.NewReader(verifyBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// 3. Повторная отправка кода подтверждения
	resendBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", strings.NewReader(resendBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// TestAuth_Verify_MaxAttempts_Resend_Verify проверяет:
// register -> многократные неверные коды до MaxAttempts -> resend-verification.
func TestAuth_Verify_MaxAttempts_Resend_Verify(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	email := "maxattempts1@example.com"

	// 1. Регистрация
	registerBody := `{"email":"` + email + `","password":"Password123!","username":"maxattempts1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// 2. Многократные неверные попытки verify до превышения лимита.
	for i := 0; i < 5; i++ { // 5 == default EMAIL_VERIFICATION_MAX_ATTEMPTS
		verifyBody := `{"email":"` + email + `","code":"000000"}`
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", strings.NewReader(verifyBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if i < 4 {
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String(), "ожидалась ошибка неверного кода до превышения лимита попыток")
		} else {
			// На последней попытке ожидаем ошибку превышения попыток (также 400).
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		}
	}

	// 3. Повторная отправка кода подтверждения должна сработать и восстановить возможность подтверждения.
	resendBody := `{"email":"` + email + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", strings.NewReader(resendBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
