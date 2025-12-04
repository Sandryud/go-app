//go:build integration
// +build integration

package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	authhandler "workout-app/internal/handler/auth"
	testcfg "workout-app/tests/integration/config"
)

// TestAuth_Register_Login_Refresh проверяет happy-path:
// регистрация -> (форсированное подтверждение email в тесте) -> логин -> refresh токенов.
func TestAuth_Register_Login_Refresh(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	// 1. Регистрация
	registerBody := `{"email":"itest1@example.com","password":"Password123!","username":"itest1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	require.Equal(t, "itest1@example.com", regResp.Email)
	require.Equal(t, "itest1", regResp.Username)
	require.NotEmpty(t, regResp.UserID)

	// В тестах код из email недоступен, поэтому мы форсируем подтверждение email в БД.
	testcfg.VerifyUserEmailForTests(t, regResp.Email)

	// 2. Логин
	loginBody := `{"email":"itest1@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	require.Equal(t, regResp.UserID, loginResp.UserID)

	// 3. Refresh
	refreshBody := `{"refresh_token":"` + loginResp.Tokens.RefreshToken + `"}`

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(refreshBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var refreshResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &refreshResp))
	require.Equal(t, loginResp.UserID, refreshResp.UserID)
	require.NotEmpty(t, refreshResp.Tokens.AccessToken)
	require.NotEmpty(t, refreshResp.Tokens.RefreshToken)
}


