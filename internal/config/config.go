package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config хранит всю конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig хранит конфигурацию сервера
type ServerConfig struct {
	Host string
	Port string
}

// DatabaseConfig хранит конфигурацию базы данных
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
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
