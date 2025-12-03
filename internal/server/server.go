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
	authhandler "workout-app/internal/handler/auth"
	"workout-app/internal/handler/health"
	"workout-app/internal/handler/middleware"
	userhandler "workout-app/internal/handler/user"
	pgrepo "workout-app/internal/repository/postgres"
	useruc "workout-app/internal/usecase/user"
	jwtsvc "workout-app/pkg/jwt"
)

// Server представляет HTTP сервер приложения
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	db         *database.DB
	cfg        *config.Config

	jwtService  jwtsvc.Service
	authHandler *authhandler.Handler
	userHandler *userhandler.Handler
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

	// Инициализируем зависимости домена пользователя и аутентификации один раз
	gormDB := db.DB
	userRepo := pgrepo.NewUserRepository(gormDB)
	userService := useruc.NewService(userRepo)
	s.jwtService = jwtsvc.NewService(&cfg.JWT)
	s.authHandler = authhandler.NewHandler(userService, userRepo, s.jwtService)
	s.userHandler = userhandler.NewHandler(userService)

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
	s.setupHealthRoutes()
	s.setupAuthRoutes()
	s.setupUserRoutes()
}

// setupHealthRoutes настраивает health-check эндпоинты.
func (s *Server) setupHealthRoutes() {
	healthHandler := health.NewHandler(s.db, s.cfg.AppEnv)
	// GET /health — базовый health-check сервера (жив ли процесс).
	s.router.GET("/health", healthHandler.Health)
	// GET /health/db — проверка доступности базы данных.
	s.router.GET("/health/db", healthHandler.HealthDB)
}

// setupAuthRoutes настраивает эндпоинты аутентификации и корневой роут API.
func (s *Server) setupAuthRoutes() {
	v1 := s.router.Group("/api/v1")

	// GET /api/v1/ — корневой эндпоинт API v1, возвращает версию и базовую информацию.
	v1.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Workout App API v1",
			"version": "1.0.0",
		})
	})

	authGroup := v1.Group("/auth")
	{
		// POST /api/v1/auth/register — регистрация нового пользователя по email/паролю/username.
		authGroup.POST("/register", s.authHandler.Register)
		// POST /api/v1/auth/login — аутентификация пользователя по email/паролю.
		authGroup.POST("/login", s.authHandler.Login)
		// POST /api/v1/auth/refresh — обновление пары access/refresh токенов по refresh-токену.
		authGroup.POST("/refresh", s.authHandler.Refresh)
	}
}

// setupUserRoutes настраивает защищённые эндпоинты пользователя.
func (s *Server) setupUserRoutes() {
	v1 := s.router.Group("/api/v1")

	userGroup := v1.Group("/users")
	userGroup.Use(middleware.Auth(s.jwtService))
	{
		// GET /api/v1/users/me — получить профиль текущего аутентифицированного пользователя.
		userGroup.GET("/me", s.userHandler.GetMe)
		// PUT /api/v1/users/me — обновить профиль текущего пользователя.
		userGroup.PUT("/me", s.userHandler.UpdateMe)
		// DELETE /api/v1/users/me — мягко удалить (деактивировать) аккаунт текущего пользователя.
		userGroup.DELETE("/me", s.userHandler.DeleteMe)
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
