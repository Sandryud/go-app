package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config хранит всю конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	CORS     CORSConfig
	JWT      JWTConfig
	Email    EmailConfig
	AppEnv   string // Окружение приложения: development, production, etc.
}

// ServerConfig хранит конфигурацию сервера
type ServerConfig struct {
	Host string
	Port string
}

// DatabaseConfig хранит конфигурацию базы данных
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int           // Максимальное количество открытых соединений
	MaxIdleConns    int           // Максимальное количество неактивных соединений
	ConnMaxLifetime time.Duration // Максимальное время жизни соединения
	ConnMaxIdleTime time.Duration // Максимальное время простоя соединения
}

// CORSConfig хранит конфигурацию CORS
type CORSConfig struct {
	AllowedOrigins   []string      // Разрешенные источники
	AllowedMethods   []string      // Разрешенные HTTP методы
	AllowedHeaders   []string      // Разрешенные заголовки
	ExposedHeaders   []string      // Заголовки, доступные клиенту
	AllowCredentials bool          // Разрешить отправку credentials
	MaxAge           time.Duration // Время кеширования preflight запросов
}

// JWTConfig хранит конфигурацию JWT-токенов (access + refresh).
type JWTConfig struct {
	AccessSecret  string        // Секрет для подписи access-токенов
	RefreshSecret string        // Секрет для подписи refresh-токенов
	AccessTTL     time.Duration // Время жизни access-токена
	RefreshTTL    time.Duration // Время жизни refresh-токена
	Issuer        string        // Issuer (iss) для токенов
}

// EmailConfig хранит конфигурацию для отправки email и параметров верификации.
type EmailConfig struct {
	SMTPHost                string        // SMTP host
	SMTPPort                int           // SMTP port
	SMTPUsername            string        // SMTP username
	SMTPPassword            string        // SMTP password
	FromEmail               string        // From email address
	VerificationTTL         time.Duration // Время жизни кода подтверждения email
	VerificationMaxAttempts int           // Максимальное количество попыток ввода кода
	VerificationCodeLength  int           // Длина кода подтверждения email
}

// DSN возвращает строку подключения к базе данных
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

// Address возвращает адрес сервера (host:port)
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	// Загружаем .env файл (если существует)
	// В production переменные окружения должны быть установлены напрямую
	_ = godotenv.Load()

	cfg := &Config{}

	// Загружаем конфигурацию сервера
	cfg.Server.Host = getEnv("SERVER_HOST", "localhost")
	cfg.Server.Port = getEnv("SERVER_PORT", "8080")

	// Загружаем конфигурацию базы данных
	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnv("DB_PORT", "5432")
	cfg.Database.User = getEnv("DB_USER", "postgres")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	cfg.Database.DBName = getEnv("DB_NAME", "workout_app")
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", "disable")

	// Загружаем настройки пула соединений
	cfg.Database.MaxOpenConns = getEnvAsInt("DB_MAX_OPEN_CONNS", 25)
	cfg.Database.MaxIdleConns = getEnvAsInt("DB_MAX_IDLE_CONNS", 5)
	cfg.Database.ConnMaxLifetime = getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	cfg.Database.ConnMaxIdleTime = getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute)

	// Загружаем окружение приложения
	cfg.AppEnv = getEnv("APP_ENV", "development")

	// Загружаем конфигурацию JWT
	cfg.JWT = JWTConfig{
		AccessSecret:  getEnv("JWT_ACCESS_SECRET", ""),
		RefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		AccessTTL:     getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:    getEnvAsDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		Issuer:        getEnv("JWT_ISSUER", "workout-app"),
	}

	// Загружаем конфигурацию Email/verification
	cfg.Email = EmailConfig{
		SMTPHost:                getEnv("EMAIL_SMTP_HOST", ""),
		SMTPPort:                getEnvAsInt("EMAIL_SMTP_PORT", 587),
		SMTPUsername:            getEnv("EMAIL_SMTP_USER", ""),
		SMTPPassword:            getEnv("EMAIL_SMTP_PASSWORD", ""),
		FromEmail:               getEnv("EMAIL_FROM", ""),
		VerificationTTL:         getEnvAsDuration("EMAIL_VERIFICATION_TTL", 15*time.Minute),
		VerificationMaxAttempts: getEnvAsInt("EMAIL_VERIFICATION_MAX_ATTEMPTS", 5),
		VerificationCodeLength:  getEnvAsInt("EMAIL_VERIFICATION_CODE_LENGTH", 6),
	}

	// Загружаем конфигурацию CORS
	cfg.CORS = loadCORSConfig(cfg.AppEnv)

	// Валидируем конфигурацию
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation error: %w", err)
	}

	return cfg, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.Server.Host == "" {
		return fmt.Errorf("SERVER_HOST must not be empty")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT must not be empty")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST must not be empty")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER must not be empty")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME must not be empty")
	}
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET must not be empty")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET must not be empty")
	}

	// Валидация email/verification настроек.
	// SMTP блок считается "выключенным", если не задан EMAIL_SMTP_HOST.
	if c.Email.SMTPHost != "" {
		if c.Email.SMTPPort <= 0 {
			return fmt.Errorf("EMAIL_SMTP_PORT must be positive")
		}
		if c.Email.SMTPUsername == "" {
			return fmt.Errorf("EMAIL_SMTP_USER must be set when EMAIL_SMTP_HOST is set")
		}
		if c.Email.SMTPPassword == "" {
			return fmt.Errorf("EMAIL_SMTP_PASSWORD must be set when EMAIL_SMTP_HOST is set")
		}
		if c.Email.FromEmail == "" {
			return fmt.Errorf("EMAIL_FROM must be set when EMAIL_SMTP_HOST is set")
		}
	}
	if c.Email.VerificationTTL <= 0 {
		return fmt.Errorf("EMAIL_VERIFICATION_TTL must be positive")
	}
	if c.Email.VerificationMaxAttempts <= 0 {
		return fmt.Errorf("EMAIL_VERIFICATION_MAX_ATTEMPTS must be positive")
	}
	if c.Email.VerificationCodeLength <= 0 {
		return fmt.Errorf("EMAIL_VERIFICATION_CODE_LENGTH must be positive")
	}
	return nil
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt получает переменную окружения как int или возвращает значение по умолчанию
func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// getEnvAsDuration получает переменную окружения как time.Duration или возвращает значение по умолчанию
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// loadCORSConfig загружает конфигурацию CORS из переменных окружения
func loadCORSConfig(appEnv string) CORSConfig {
	// Значения по умолчанию для development
	defaultOrigins := []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:8080",
	}
	defaultMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	defaultHeaders := []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"X-Requested-With",
		"Accept",
		"Accept-Encoding",
		"X-CSRF-Token",
	}
	defaultExposedHeaders := []string{"Content-Length", "Content-Type", "Authorization"}

	cfg := CORSConfig{
		AllowedOrigins:   getEnvAsSlice("CORS_ALLOWED_ORIGINS", defaultOrigins),
		AllowedMethods:   getEnvAsSlice("CORS_ALLOWED_METHODS", defaultMethods),
		AllowedHeaders:   getEnvAsSlice("CORS_ALLOWED_HEADERS", defaultHeaders),
		ExposedHeaders:   getEnvAsSlice("CORS_EXPOSED_HEADERS", defaultExposedHeaders),
		AllowCredentials: getEnv("CORS_ALLOW_CREDENTIALS", "true") == "true",
		MaxAge:           getEnvAsDuration("CORS_MAX_AGE", 12*time.Hour),
	}

	// В production, если не указаны origins, используем пустой список (более безопасно)
	if appEnv == "production" && os.Getenv("CORS_ALLOWED_ORIGINS") == "" {
		cfg.AllowedOrigins = []string{}
	}

	return cfg
}

// getEnvAsSlice получает переменную окружения как slice строк, разделенных запятыми
func getEnvAsSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Разделяем по запятой и очищаем пробелы
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
