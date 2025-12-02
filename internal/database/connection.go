package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"workout-app/internal/config"
)

// Константы для значений по умолчанию пула соединений
const (
	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 5 * time.Minute
	defaultConnMaxIdleTime = 10 * time.Minute
)

// DB представляет подключение к базе данных
type DB struct {
	*gorm.DB
}

// NewConnection создает новое подключение к базе данных.
// Принимает конфигурацию базы данных и окружение приложения для настройки логирования.
// Возвращает инициализированное подключение или ошибку в случае неудачи.
//
// Пример использования:
//
//	db, err := database.NewConnection(cfg.Database, cfg.AppEnv)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
func NewConnection(cfg *config.DatabaseConfig, appEnv string) (*DB, error) {
	// Валидация входных параметров
	if cfg == nil {
		return nil, fmt.Errorf("конфигурация базы данных не может быть nil")
	}

	log.Println("Инициализация подключения к базе данных...")

	// Настройка уровня логирования GORM в зависимости от окружения
	gormLogger := logger.Default
	if strings.ToLower(appEnv) == "development" {
		// В development режиме используем более подробное логирование
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	// Создаем подключение к базе данных
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	// Получаем sql.DB для настройки пула соединений
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения sql.DB: %w", err)
	}

	// Настраиваем пул соединений из конфигурации
	// Используем значения из конфига, если они заданы, иначе значения по умолчанию
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = defaultMaxOpenConns
	}
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = defaultMaxIdleConns
	}
	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime == 0 {
		connMaxLifetime = defaultConnMaxLifetime
	}
	connMaxIdleTime := cfg.ConnMaxIdleTime
	if connMaxIdleTime == 0 {
		connMaxIdleTime = defaultConnMaxIdleTime
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	// Проверяем подключение
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки подключения к базе данных: %w", err)
	}

	log.Println("Подключение к базе данных установлено успешно")

	return &DB{DB: db}, nil
}

// Close закрывает подключение к базе данных.
// Освобождает все ресурсы, связанные с подключением.
// Возвращает ошибку в случае неудачи при закрытии.
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения sql.DB для закрытия: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия подключения к базе данных: %w", err)
	}

	log.Println("Подключение к базе данных закрыто")
	return nil
}

// Ping проверяет доступность базы данных.
// Используется для health checks и проверки работоспособности подключения.
// Возвращает ошибку, если база данных недоступна.
func (db *DB) Ping() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ошибка ping базы данных: %w", err)
	}

	return nil
}
