package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config хранит всю конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
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

	// Валидируем конфигурацию
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return cfg, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.Server.Host == "" {
		return fmt.Errorf("SERVER_HOST не может быть пустым")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT не может быть пустым")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST не может быть пустым")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER не может быть пустым")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME не может быть пустым")
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
