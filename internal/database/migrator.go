package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq" // PostgreSQL driver

	"workout-app/internal/config"
	"workout-app/internal/database/migrations"
)

var (
	// ErrNoChange возвращается, когда нет миграций для применения.
	ErrNoChange = errors.New("no change")

	// ErrInvalidVersion возвращается, когда указана некорректная версия миграции.
	ErrInvalidVersion = errors.New("invalid version")

	// ErrDirtyState возвращается, когда миграции находятся в "грязном" состоянии.
	// Это означает, что миграция была прервана и требует ручного вмешательства.
	ErrDirtyState = errors.New("database is in dirty state")
)

// Migrator предоставляет функционал для управления миграциями базы данных.
// Использует библиотеку golang-migrate для управления версиями схемы БД.
type Migrator struct {
	m *migrate.Migrate
}

// NewMigrator создает новый экземпляр мигратора.
// Принимает подключение к базе данных и создает экземпляр migrate.Migrate
// с использованием встроенных SQL файлов миграций.
//
// Пример использования:
//
//	migrator, err := database.NewMigrator(db)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer migrator.Close()
//
//	if err := migrator.Up(); err != nil {
//	    log.Fatal(err)
//	}
func NewMigrator(db *DB) (*Migrator, error) {
	// Получаем sql.DB из GORM
	sqlDB, err := db.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения sql.DB: %w", err)
	}

	// Создаем драйвер для PostgreSQL
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания драйвера PostgreSQL: %w", err)
	}

	// Создаем источник миграций из embed.FS
	source, err := iofs.New(migrations.Migrations, ".")
	if err != nil {
		return nil, fmt.Errorf("ошибка создания источника миграций: %w", err)
	}

	// Создаем экземпляр migrate.Migrate
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания экземпляра migrate: %w", err)
	}

	return &Migrator{m: m}, nil
}

// NewMigratorFromDSN создает новый экземпляр мигратора напрямую из DSN строки.
// Полезно для случаев, когда нужно создать отдельное подключение для миграций.
func NewMigratorFromDSN(dsn string) (*Migrator, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия подключения: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка создания драйвера PostgreSQL: %w", err)
	}

	source, err := iofs.New(migrations.Migrations, ".")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка создания источника миграций: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка создания экземпляра migrate: %w", err)
	}

	return &Migrator{m: m}, nil
}

// NewMigratorFromConfig создает новый экземпляр мигратора из конфигурации базы данных.
func NewMigratorFromConfig(cfg *config.DatabaseConfig) (*Migrator, error) {
	return NewMigratorFromDSN(cfg.DSN())
}

// Close закрывает подключение мигратора и освобождает ресурсы.
func (m *Migrator) Close() error {
	if m.m == nil {
		return nil
	}
	sourceErr, dbErr := m.m.Close()
	if sourceErr != nil {
		return fmt.Errorf("ошибка закрытия источника миграций: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("ошибка закрытия подключения к БД: %w", dbErr)
	}
	return nil
}

// Up применяет все доступные миграции вверх (forward).
// Возвращает ErrNoChange, если нет миграций для применения.
func (m *Migrator) Up() error {
	if err := m.m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrNoChange
		}
		return fmt.Errorf("ошибка применения миграций: %w", err)
	}
	log.Println("Все миграции успешно применены")
	return nil
}

// Down откатывает последнюю примененную миграцию.
// Возвращает ErrNoChange, если нет миграций для отката.
func (m *Migrator) Down() error {
	if err := m.m.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrNoChange
		}
		return fmt.Errorf("ошибка отката миграции: %w", err)
	}
	log.Println("Миграция успешно откатилась")
	return nil
}

// Steps применяет или откатывает N миграций в зависимости от знака.
// Положительное число применяет миграции вверх, отрицательное - откатывает вниз.
func (m *Migrator) Steps(n int) error {
	if err := m.m.Steps(n); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrNoChange
		}
		direction := "вверх"
		if n < 0 {
			direction = "вниз"
		}
		return fmt.Errorf("ошибка применения %d миграций %s: %w", n, direction, err)
	}
	log.Printf("Успешно применено %d миграций\n", n)
	return nil
}

// Version возвращает текущую версию базы данных и флаг "грязного" состояния.
// Возвращает (version, dirty, error).
// Если миграции не применялись, версия будет 0 и dirty = false.
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("ошибка получения версии: %w", err)
	}
	return version, dirty, nil
}

// Force устанавливает версию миграции без применения миграций.
// Используется для восстановления после "грязного" состояния.
// ВНИМАНИЕ: Используйте только в критических ситуациях!
func (m *Migrator) Force(version int) error {
	if err := m.m.Force(version); err != nil {
		return fmt.Errorf("ошибка принудительной установки версии %d: %w", version, err)
	}
	log.Printf("Версия миграции принудительно установлена на %d\n", version)
	return nil
}

// CheckDirty проверяет, находится ли база данных в "грязном" состоянии.
// Возвращает true, если требуется ручное вмешательство.
func (m *Migrator) CheckDirty() (bool, error) {
	_, dirty, err := m.Version()
	if err != nil {
		return false, err
	}
	if dirty {
		return true, ErrDirtyState
	}
	return false, nil
}
