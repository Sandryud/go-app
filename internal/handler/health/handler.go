package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"workout-app/internal/database"
)

// Handler обрабатывает health check запросы
type Handler struct {
	db     *database.DB
	appEnv string
}

// NewHandler создает новый экземпляр health handler
func NewHandler(db *database.DB, appEnv string) *Handler {
	return &Handler{
		db:     db,
		appEnv: appEnv,
	}
}

// HealthResponse представляет ответ health check
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Health проверяет работоспособность сервера
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Message: "Сервер работает",
	})
}

// HealthDB проверяет подключение к базе данных
func (h *Handler) HealthDB(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:  "error",
			Message: "База данных не инициализирована",
		})
		return
	}

	// Создаем контекст с таймаутом для health check (5 секунд)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Проверяем подключение к БД с контекстом
	// Используем канал для выполнения Ping в отдельной горутине с таймаутом
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.db.Ping()
	}()

	var err error
	select {
	case err = <-errCh:
		// Ping завершился
	case <-ctx.Done():
		// Таймаут
		err = ctx.Err()
	}

	if err != nil {
		// Определяем сообщение об ошибке в зависимости от окружения
		errorMessage := "База данных недоступна"
		if h.appEnv != "production" {
			// В development показываем детали ошибки
			errorMessage = "База данных недоступна: " + err.Error()
		}

		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:  "error",
			Message: errorMessage,
		})
		return
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Message: "База данных доступна",
	})
}
