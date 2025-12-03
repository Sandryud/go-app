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
	registerBody := `{"email":"u_flow@example.com","password":"Password123!","username":"u_flow"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp authhandler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	access := regResp.Tokens.AccessToken

	// 2. GET /users/me
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var profile userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &profile))
	require.Equal(t, "u_flow", profile.Username)

	// 3. PUT /users/me (обновление профиля)
	updateBody := `{"username":"u_flow_new","first_name":"Test","training_level":"intermediate"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/me", strings.NewReader(updateBody))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var updated userhandler.ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	require.Equal(t, "u_flow_new", updated.Username)
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


