package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"workout-app/internal/config"
)

// CORS middleware для настройки Cross-Origin Resource Sharing
// Принимает конфигурацию CORS и настраивает middleware соответственно
func CORS(cfg *config.CORSConfig) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowMethods:     cfg.AllowedMethods,
		AllowHeaders:     cfg.AllowedHeaders,
		ExposeHeaders:    cfg.ExposedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}

	// В development режиме разрешаем все источники, если список пуст
	// В production используем только явно указанные источники
	if gin.Mode() == gin.DebugMode && len(cfg.AllowedOrigins) == 0 {
		corsConfig.AllowAllOrigins = true
	} else if len(cfg.AllowedOrigins) > 0 {
		// Используем AllowOrigins, если указаны конкретные источники
		corsConfig.AllowOrigins = cfg.AllowedOrigins
	} else {
		// В production, если origins не указаны, блокируем все
		corsConfig.AllowOrigins = []string{}
	}

	return cors.New(corsConfig)
}
