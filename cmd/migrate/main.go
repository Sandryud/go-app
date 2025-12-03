package main

import (
	"log"
	"os"

	"workout-app/internal/config"
	"workout-app/internal/database"
)

// migrateUsers создает таблицу users и связанные объекты (индексы, триггеры).
func migrateUsers(db *database.DB) error {
	sqlBytes, err := os.ReadFile("internal/database/migrations/001_create_users_table.sql")
	if err != nil {
		return err
	}

	return db.Exec(string(sqlBytes)).Error
}

func main() {
	log.Println("Запуск миграции базы данных...")

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем подключение к базе данных
	db, err := database.NewConnection(&cfg.Database, cfg.AppEnv)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка закрытия подключения к базе данных: %v", err)
		}
	}()

	// Применяем миграции
	if err := migrateUsers(db); err != nil {
		log.Fatalf("Ошибка применения миграции users: %v", err)
	}

	log.Println("Миграции успешно применены")
}


