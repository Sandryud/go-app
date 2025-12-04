package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery middleware для обработки паник и предотвращения краша приложения
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Логируем панику с подробной информацией
		log.Printf("Паника перехвачена: %v\n", recovered)

		// Логируем стек вызовов (если доступен)
		if err, ok := recovered.(string); ok {
			log.Printf("Ошибка: %s\n", err)
		}

		// Возвращаем 500 ошибку клиенту
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Внутренняя ошибка сервера",
			"message": "Произошла непредвиденная ошибка. Пожалуйста, попробуйте позже.",
		})

		c.Abort()
	})
}

// RecoveryWithLogger - расширенная версия с дополнительным логированием
func RecoveryWithLogger() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		// Получаем информацию о запросе
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		// Логируем панику с контекстом
		log.Printf("[PANIC] %s %s from %s: %v\n", method, path, clientIP, recovered)

		// В production режиме не показываем детали ошибки
		errorMessage := "Внутренняя ошибка сервера"
		if gin.Mode() == gin.DebugMode {
			errorMessage = fmt.Sprintf("%v", recovered)
		}

		// Возвращаем ошибку
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": errorMessage,
		})

		c.Abort()
	})
}
