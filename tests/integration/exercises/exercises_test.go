//go:build integration
// +build integration

package exercises_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	testcfg "workout-app/tests/integration/config"
)

const exercisesCatalogEnv = "EXERCISES_CATALOG_PATH"

// getAccessToken регистрирует пользователя, подтверждает email и логинится; возвращает access token.
func getAccessToken(t *testing.T, router interface{ ServeHTTP(http.ResponseWriter, *http.Request) }) string {
	t.Helper()
	email := "exercises-test@example.com"
	password := "Password123!"
	username := "exercisestest"

	registerBody := `{"email":"` + email + `","password":"` + password + `","username":"` + username + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	// Может быть 201 или 409 если пользователь уже есть от предыдущего теста
	if w.Code != http.StatusCreated && w.Code != http.StatusConflict {
		t.Fatalf("register: want 201 or 409, got %d: %s", w.Code, w.Body.String())
	}

	testcfg.VerifyUserEmailForTests(t, email)

	loginBody := `{"email":"` + email + `","password":"` + password + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp testcfg.LoginResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	require.NotEmpty(t, loginResp.Tokens.AccessToken)
	return loginResp.Tokens.AccessToken
}

func TestExercises_GetVersion_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exercises.json")
	body := []byte(`{"meta":{"version":"test-v1"},"exercises":[]}`)
	require.NoError(t, os.WriteFile(path, body, 0644))

	oldVal := os.Getenv(exercisesCatalogEnv)
	os.Setenv(exercisesCatalogEnv, path)
	t.Cleanup(func() { os.Setenv(exercisesCatalogEnv, oldVal) })

	router := testcfg.NewTestRouter(t)
	accessToken := getAccessToken(t, router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/version", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	var resp struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "test-v1", resp.Version)
}

func TestExercises_GetData_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exercises.json")
	body := []byte(`{"meta":{"version":"test-v2"},"exercises":[]}`)
	require.NoError(t, os.WriteFile(path, body, 0644))

	oldVal := os.Getenv(exercisesCatalogEnv)
	os.Setenv(exercisesCatalogEnv, path)
	t.Cleanup(func() { os.Setenv(exercisesCatalogEnv, oldVal) })

	router := testcfg.NewTestRouter(t)
	accessToken := getAccessToken(t, router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.Equal(t, `"test-v2"`, w.Header().Get("ETag"))
	require.JSONEq(t, string(body), w.Body.String())
}

func TestExercises_GetData_304_WhenETagMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exercises.json")
	body := []byte(`{"meta":{"version":"v304"},"exercises":[]}`)
	require.NoError(t, os.WriteFile(path, body, 0644))

	oldVal := os.Getenv(exercisesCatalogEnv)
	os.Setenv(exercisesCatalogEnv, path)
	t.Cleanup(func() { os.Setenv(exercisesCatalogEnv, oldVal) })

	router := testcfg.NewTestRouter(t)
	accessToken := getAccessToken(t, router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("If-None-Match", `"v304"`)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotModified, w.Code)
	require.Empty(t, w.Body.Bytes())
}

func TestExercises_GetData_404_WhenCatalogNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	oldVal := os.Getenv(exercisesCatalogEnv)
	os.Setenv(exercisesCatalogEnv, path)
	t.Cleanup(func() { os.Setenv(exercisesCatalogEnv, oldVal) })

	router := testcfg.NewTestRouter(t)
	accessToken := getAccessToken(t, router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "not_found", resp.Error.Code)
}

func TestExercises_Unauthorized_WithoutToken(t *testing.T) {
	router := testcfg.NewTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/version", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestExercises_GetVersion_500_WhenCatalogNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	oldVal := os.Getenv(exercisesCatalogEnv)
	os.Setenv(exercisesCatalogEnv, path)
	t.Cleanup(func() { os.Setenv(exercisesCatalogEnv, oldVal) })

	router := testcfg.NewTestRouter(t)
	accessToken := getAccessToken(t, router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/version", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "internal_error", resp.Error.Code)
}
