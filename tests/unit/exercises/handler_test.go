package exercises_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	exerciseshandler "workout-app/internal/handler/exercises"
	repo "workout-app/internal/repository/interfaces"
	"workout-app/pkg/logger"
)

// fakeExercisesService реализует usecase для тестов handler'а.
type fakeExercisesService struct {
	version    string
	versionErr error
	data       []byte
	dataVer    string
	dataErr    error
}

func (f *fakeExercisesService) GetVersion(_ context.Context) (string, error) {
	return f.version, f.versionErr
}

func (f *fakeExercisesService) GetData(_ context.Context) ([]byte, string, error) {
	return f.data, f.dataVer, f.dataErr
}

func TestHandler_GetVersion_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeExercisesService{version: "20260227102513"}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/exercises/version", nil)

	h.GetVersion(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	var body struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "20260227102513", body.Version)
}

func TestHandler_GetVersion_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeExercisesService{versionErr: context.DeadlineExceeded}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/exercises/version", nil)

	h.GetVersion(c)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "internal_error", body.Error.Code)
}

func TestHandler_GetData_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawJSON := []byte(`{"meta":{"version":"v123"},"exercises":[]}`)
	uc := &fakeExercisesService{data: rawJSON, dataVer: "v123"}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)

	h.GetData(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.Equal(t, `"v123"`, w.Header().Get("ETag"))
	require.Equal(t, rawJSON, w.Body.Bytes())
}

func TestHandler_GetData_304_WhenETagMatches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawJSON := []byte(`{"meta":{"version":"v123"}}`)
	uc := &fakeExercisesService{data: rawJSON, dataVer: "v123"}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	engine := gin.New()
	engine.GET("/api/v1/exercises/data", h.GetData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)
	req.Header.Set("If-None-Match", `"v123"`)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotModified, w.Code)
	require.Empty(t, w.Body.Bytes())
}

func TestHandler_GetData_304_WhenETagMatchesWithWPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeExercisesService{data: []byte(`{}`), dataVer: "v456"}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	engine := gin.New()
	engine.GET("/api/v1/exercises/data", h.GetData)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)
	req.Header.Set("If-None-Match", `W/"v456"`)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotModified, w.Code)
}

func TestHandler_GetData_200_WhenETagDoesNotMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawJSON := []byte(`{"meta":{"version":"v789"}}`)
	uc := &fakeExercisesService{data: rawJSON, dataVer: "v789"}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)
	req.Header.Set("If-None-Match", `"old-version"`)
	c.Request = req

	h.GetData(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, rawJSON, w.Body.Bytes())
}

func TestHandler_GetData_404_WhenNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeExercisesService{dataErr: repo.ErrExercisesNotFound}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)

	h.GetData(c)

	require.Equal(t, http.StatusNotFound, w.Code)
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "not_found", body.Error.Code)
}

func TestHandler_GetData_500_OnOtherError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeExercisesService{dataErr: context.DeadlineExceeded}
	h := exerciseshandler.NewHandler(uc, logger.Default())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/exercises/data", nil)

	h.GetData(c)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "internal_error", body.Error.Code)
}
