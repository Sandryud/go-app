//go:build integration
// +build integration

package config

import (
	"workout-app/internal/database"
)

// MigrateDatabase применяет все миграции к тестовой базе данных.
// Используется в интеграционных тестах для инициализации схемы БД.
// ВАЖНО: Не закрывает мигратор, так как соединение используется дальше в тестах.
func MigrateDatabase(db *database.DB) error {
	migrator, err := database.NewMigrator(db)
	if err != nil {
		return err
	}
	// НЕ закрываем мигратор здесь, так как мы используем то же соединение db дальше в тестах.
	// При использовании WithInstance, мигратор не владеет соединением,
	// но на всякий случай не вызываем Close() чтобы избежать проблем.

	// Применяем все миграции
	// Игнорируем ErrNoChange, так как миграции могут быть уже применены
	if err := migrator.Up(); err != nil && err != database.ErrNoChange {
		return err
	}

	return nil
}
