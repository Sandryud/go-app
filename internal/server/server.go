package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"workout-app/internal/config"
	"workout-app/internal/database"
	"workout-app/internal/handler/health"
	"workout-app/internal/handler/middleware"
)

// Server представляет HTTP сервер приложения
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	db         *database.DB
	cfg        *config.Config
}

// NewServer создает новый экземпляр сервера
func NewServer(cfg *config.Config, db *database.DB) *Server {
	// Устанавливаем режим Gin в зависимости от окружения
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	s := &Server{
		router: router,
		db:     db,
		cfg:    cfg,
	}

	// Настраиваем middleware и роуты
	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware настраивает middleware для роутера
func (s *Server) setupMiddleware() {
	// Recovery middleware - должен быть первым для перехвата паник
	s.router.Use(middleware.Recovery())

	// Logger middleware - логирование всех запросов
	s.router.Use(middleware.LoggerStructured())

	// CORS middleware - настройка CORS
	s.router.Use(middleware.CORS(&s.cfg.CORS))
}

// setupRoutes настраивает маршруты приложения
func (s *Server) setupRoutes() {
	// Health check endpoints
	healthHandler := health.NewHandler(s.db, s.cfg.AppEnv)
	s.router.GET("/health", healthHandler.Health)
	s.router.GET("/health/db", healthHandler.HealthDB)

	// API v1 группа
	v1 := s.router.Group("/api/v1")
	{
		// Здесь будут добавлены роуты для различных доменов
		// Например: v1.GET("/users", userHandler.GetUsers)
		v1.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Workout App API v1",
				"version": "1.0.0",
			})
		})
	}
}

// Start запускает HTTP сервер с graceful shutdown
func (s *Server) Start() error {
	address := s.cfg.Server.Address()

	s.httpServer = &http.Server{
		Addr:           address,
		Handler:        s.router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Канал для получения сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Канал для ошибок запуска сервера
	serverErr := make(chan error, 1)

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("HTTP сервер запущен на %s", address)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("ошибка запуска HTTP сервера: %w", err)
		}
	}()

	// Ожидаем либо сигнал для graceful shutdown, либо ошибку запуска
	select {
	case err := <-serverErr:
		// Если сервер не смог запуститься, пытаемся корректно остановить
		log.Printf("Ошибка запуска сервера: %v", err)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(ctx)
		return err
	case sig := <-quit:
		log.Printf("Получен сигнал %v для остановки сервера...", sig)
	}

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем сервер
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("ошибка при остановке сервера: %w", err)
	}

	log.Println("HTTP сервер успешно остановлен")
	return nil
}

// GetRouter возвращает роутер (для тестирования)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
