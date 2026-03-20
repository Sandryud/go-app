//go:build integration
// +build integration

package config

import (
	"workout-app/internal/database"
)

// MigrateDatabase применяет все миграции к тестовой базе данных.
// Используется в интеграционных тестах для инициализации схемы БД.
//
// При dirty очищаем schema_migrations целиком — следующий Up() применит все миграции
// заново (SQL в проекте идемпотентен через IF NOT EXISTS / ADD COLUMN IF NOT EXISTS).
// Это чинит и рассинхрон «версия N в таблице, а объекты схемы отсутствуют».
//
// Строка version=0 (например после migrate -force 0) ломает Up() с ошибкой
// «read down for version 0» — перед Up удаляем такие строки (как make migrate-fix-dirty-0).
//
// Не вызываем migrator.Close(): драйвер golang-migrate закрывает тот же *sql.DB, что и GORM.
func MigrateDatabase(db *database.DB) error {
	if err := resolveTestSchemaDirty(db); err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM schema_migrations WHERE version = 0").Error; err != nil {
		return err
	}

	migrator, err := database.NewMigrator(db)
	if err != nil {
		return err
	}

	if err := migrator.Up(); err != nil && err != database.ErrNoChange {
		return err
	}
	return nil
}

func resolveTestSchemaDirty(db *database.DB) error {
	migrator, err := database.NewMigrator(db)
	if err != nil {
		return err
	}

	_, dirty, err := migrator.Version()
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	return db.Exec("DELETE FROM schema_migrations").Error
}
