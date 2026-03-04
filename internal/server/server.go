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

	_ "workout-app/api/swagger" // docs
	"workout-app/internal/config"
	"workout-app/internal/database"
	domain "workout-app/internal/domain/user"
	authhandler "workout-app/internal/handler/auth"
	exerciseshandler "workout-app/internal/handler/exercises"
	"workout-app/internal/handler/health"
	"workout-app/internal/handler/middleware"
	planshandler "workout-app/internal/handler/plans"
	userhandler "workout-app/internal/handler/user"
	"workout-app/internal/mailer"
	filerepo "workout-app/internal/repository/file"
	pgrepo "workout-app/internal/repository/postgres"
	authuc "workout-app/internal/usecase/auth"
	exercisesuc "workout-app/internal/usecase/exercises"
	plansuc "workout-app/internal/usecase/plans"
	useruc "workout-app/internal/usecase/user"
	"workout-app/pkg/jwt"
	"workout-app/pkg/logger"
	mailerpkg "workout-app/pkg/mailer"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server представляет HTTP сервер приложения
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	db         *database.DB
	cfg        *config.Config

	logger           logger.Logger
	jwtService       jwt.Service
	authHandler      *authhandler.Handler
	userHandler      *userhandler.Handler
	exercisesHandler *exerciseshandler.Handler
	plansHandler     *planshandler.Handler
}

// loggerEmailSender — простая реализация EmailSender, логирующая коды в логгер.
// Используется как временное решение до внедрения полноценного почтового сервиса.
type loggerEmailSender struct {
	logger logger.Logger
}

func (s *loggerEmailSender) SendEmailVerificationCode(ctx context.Context, email, code string) error {
	s.logger.Info("Email verification code sent", map[string]any{
		"email": email,
		"code":  code,
	})
	return nil
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

	s.logger = logger.Default()

	// Инициализируем зависимости домена пользователя и аутентификации один раз
	gormDB := db.DB
	userRepo := pgrepo.NewUserRepository(gormDB)
	emailVerifRepo := pgrepo.NewEmailVerificationRepository(gormDB)
	s.jwtService = jwt.NewService(&cfg.JWT)

	var emailSender mailerpkg.EmailSender
	if cfg.Email.SMTPHost != "" {
		emailSender = mailer.NewSMTPSender(&cfg.Email, s.logger)
	} else {
		// Фолбэк: логируем коды в лог вместо реальной отправки писем.
		emailSender = &loggerEmailSender{logger: s.logger}
	}

	authService := authuc.NewService(
		userRepo,
		emailVerifRepo,
		s.jwtService,
		emailSender,
		cfg.Email.VerificationTTL,
		cfg.Email.VerificationMaxAttempts,
		cfg.Email.VerificationCodeLength,
	)

	// userService использует тот же emailSender и jwtService, что и authService
	userService := useruc.NewService(
		userRepo,
		emailVerifRepo,
		emailSender,
		s.jwtService,
		cfg.Email.VerificationTTL,
		cfg.Email.VerificationMaxAttempts,
		cfg.Email.VerificationCodeLength,
	)

	s.authHandler = authhandler.NewHandler(authService, s.logger)
	s.userHandler = userhandler.NewHandler(userService, s.logger)

	exercisesRepo := filerepo.NewExercisesRepository(cfg.ExercisesCatalogPath)
	exercisesService := exercisesuc.NewService(exercisesRepo)
	s.exercisesHandler = exerciseshandler.NewHandler(exercisesService, s.logger)

	planRepo := pgrepo.NewPlanRepository(gormDB)
	plansService := plansuc.NewService(planRepo, exercisesRepo)
	s.plansHandler = planshandler.NewHandler(plansService, s.logger)

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
	s.setupExercisesRoutes()
	s.setupPlansRoutes()

	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
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
		// POST /api/v1/auth/verify-email — подтверждение email одноразовым кодом.
		authGroup.POST("/verify-email", s.authHandler.VerifyEmail)
		// POST /api/v1/auth/resend-verification — повторная отправка кода подтверждения email.
		authGroup.POST("/resend-verification", s.authHandler.ResendVerification)
		// POST /api/v1/auth/refresh — обновление пары access/refresh токенов по refresh-токену.
		authGroup.POST("/refresh", s.authHandler.Refresh)
		// POST /api/v1/auth/restore — восстановить удалённый аккаунт по email и паролю.
		authGroup.POST("/restore", s.authHandler.RestoreAccount)
	}
}

// setupUserRoutes настраивает защищённые эндпоинты пользователя.
func (s *Server) setupUserRoutes() {
	v1 := s.router.Group("/api/v1")

	userGroup := v1.Group("/users")
	userGroup.Use(middleware.Auth(s.jwtService, s.logger))
	{
		// GET /api/v1/users/me — получить профиль текущего аутентифицированного пользователя.
		userGroup.GET("/me", s.userHandler.GetMe)
		// PUT /api/v1/users/me — обновить профиль текущего пользователя.
		userGroup.PUT("/me", s.userHandler.UpdateMe)
		// DELETE /api/v1/users/me — мягко удалить (деактивировать) аккаунт текущего пользователя.
		userGroup.DELETE("/me", s.userHandler.DeleteMe)
		// POST /api/v1/users/me/change-email — запросить изменение email (отправка кода на новый email).
		userGroup.POST("/me/change-email", s.userHandler.RequestEmailChange)
		// POST /api/v1/users/me/verify-email-change — подтвердить изменение email по коду.
		userGroup.POST("/me/verify-email-change", s.userHandler.VerifyEmailChange)
		// GET /api/v1/users/:id — получить публичный профиль пользователя по ID.
		userGroup.GET("/:id", s.userHandler.GetByID)
	}

	// Админские роуты
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.Auth(s.jwtService, s.logger), middleware.RequireRole(s.logger, domain.RoleAdmin))
	{
		// GET /api/v1/admin/users — список всех активных пользователей (только для admin).
		adminGroup.GET("/users", s.userHandler.ListUsers)
	}
}

// setupExercisesRoutes настраивает эндпоинты каталога упражнений (только для авторизованных пользователей).
func (s *Server) setupExercisesRoutes() {
	v1 := s.router.Group("/api/v1")
	exercisesGroup := v1.Group("/exercises")
	exercisesGroup.Use(middleware.Auth(s.jwtService, s.logger))
	{
		// GET /api/v1/exercises/version — версия каталога упражнений.
		exercisesGroup.GET("/version", s.exercisesHandler.GetVersion)
		// GET /api/v1/exercises/data — полный каталог (JSON), поддерживает ETag и 304.
		exercisesGroup.GET("/data", s.exercisesHandler.GetData)
	}
}

// setupPlansRoutes настраивает эндпоинты планов тренировок (требуется авторизация).
func (s *Server) setupPlansRoutes() {
	v1 := s.router.Group("/api/v1")
	plansGroup := v1.Group("/plans")
	plansGroup.Use(middleware.Auth(s.jwtService, s.logger))
	{
		// GET /api/v1/plans — список планов текущего пользователя.
		plansGroup.GET("", s.plansHandler.List)
		// GET /api/v1/plans/:id — один план с деревом дней и упражнений.
		plansGroup.GET("/:id", s.plansHandler.GetByID)
		// POST /api/v1/plans — создать план.
		plansGroup.POST("", s.plansHandler.Create)
		// PUT /api/v1/plans/:id — обновить план.
		plansGroup.PUT("/:id", s.plansHandler.Update)
		// DELETE /api/v1/plans/:id — удалить план.
		plansGroup.DELETE("/:id", s.plansHandler.Delete)

		// Дни плана
		plansGroup.POST("/:planId/days", s.plansHandler.AddDay)
		plansGroup.PUT("/:planId/days/:dayId", s.plansHandler.UpdateDay)
		plansGroup.DELETE("/:planId/days/:dayId", s.plansHandler.DeleteDay)

		// Упражнения в дне
		plansGroup.POST("/:planId/days/:dayId/exercises", s.plansHandler.AddExerciseToDay)
		plansGroup.PUT("/:planId/days/:dayId/exercises/:exerciseEntryId", s.plansHandler.UpdateExerciseInDay)
		plansGroup.DELETE("/:planId/days/:dayId/exercises/:exerciseEntryId", s.plansHandler.DeleteExerciseFromDay)
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
