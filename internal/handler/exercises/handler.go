package exercises

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"workout-app/internal/handler/response"
	exercisesuc "workout-app/internal/usecase/exercises"
	"workout-app/pkg/logger"
)

// Handler обрабатывает HTTP-запросы каталога упражнений.
type Handler struct {
	uc    exercisesuc.Service
	logger logger.Logger
}

// NewHandler создаёт новый handler каталога упражнений.
func NewHandler(uc exercisesuc.Service, logger logger.Logger) *Handler {
	return &Handler{
		uc:     uc,
		logger: logger,
	}
}

// GetVersion godoc
// @Summary      Версия каталога упражнений
// @Description  Возвращает текущую версию каталога (meta.version из exercises.json). Требуется авторизация.
// @Tags         exercises
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  VersionResponse
// @Failure      401  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/exercises/version [get]
func (h *Handler) GetVersion(c *gin.Context) {
	version, err := h.uc.GetVersion(c.Request.Context())
	if err != nil {
		h.logger.Error("exercises_get_version_error", map[string]any{
			"error": err.Error(),
			"path":  c.Request.URL.Path,
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Failed to get exercises catalog version", nil)
		return
	}
	c.JSON(http.StatusOK, VersionResponse{Version: version})
}

// GetData godoc
// @Summary      Данные каталога упражнений
// @Description  Возвращает полный файл exercises.json. Поддерживает ETag: при заголовке If-None-Match, совпадающем с текущей версией, возвращается 304 Not Modified. Требуется авторизация.
// @Tags         exercises
// @Security     BearerAuth
// @Produce      json
// @Param        If-None-Match  header    string  false  "ETag версии каталога для условного запроса"
// @Success      200  {string}  string  "Полный JSON каталога (meta + exercises)"
// @Success      304  {string}  string  "Not Modified — версия клиента совпадает с текущей, тело пустое"
// @Failure      401  {object}  response.ErrorBody
// @Failure      404  {object}  response.ErrorBody  "Файл каталога не найден"
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/exercises/data [get]
func (h *Handler) GetData(c *gin.Context) {
	rawJSON, version, err := h.uc.GetData(c.Request.Context())
	if err != nil {
		if errors.Is(err, exercisesuc.ErrCatalogNotFound) {
			response.Error(c, http.StatusNotFound, "not_found", "Exercises catalog not found", nil)
			return
		}
		h.logger.Error("exercises_get_data_error", map[string]any{
			"error": err.Error(),
			"path":  c.Request.URL.Path,
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Failed to read exercises catalog", nil)
		return
	}

	// Проверка If-None-Match для 304
	if noneMatch := c.GetHeader("If-None-Match"); noneMatch != "" {
		if etagMatches(version, noneMatch) {
			c.Status(http.StatusNotModified)
			return
		}
	}

	c.Header("Content-Type", "application/json")
	c.Header("ETag", `"`+version+`"`)
	c.Data(http.StatusOK, "application/json", rawJSON)
}

// etagMatches проверяет, совпадает ли текущая версия с одним из значений If-None-Match.
// Учитывает формат ETag (с кавычками и опциональным префиксом W/).
func etagMatches(version, ifNoneMatch string) bool {
	for _, s := range strings.Split(ifNoneMatch, ",") {
		s = strings.TrimSpace(s)
		s = strings.TrimPrefix(s, "W/")
		s = strings.Trim(s, `"`)
		if s == version {
			return true
		}
	}
	return false
}
