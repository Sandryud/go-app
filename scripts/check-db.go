package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"workout-app/internal/config"
	"workout-app/internal/database"
)

// fileExists проверяет существование файла
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func main() {
	log.Println("Проверка подключения к базе данных...")

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Если скрипт запущен на хосте (не в Docker), заменяем "postgres" на "localhost"
	// Проверяем, находимся ли мы внутри Docker контейнера
	// Внутри Docker контейнера обычно есть файл /.dockerenv или переменная окружения указывает на Docker
	isInDocker := os.Getenv("container") != "" || fileExists("/.dockerenv")
	if cfg.Database.Host == "postgres" && !isInDocker {
		log.Println("⚠️  Обнаружен хост 'postgres', но скрипт запущен вне Docker")
		log.Println("   Автоматически изменяю DB_HOST на 'localhost' для локального подключения")
		cfg.Database.Host = "localhost"
	}

	// Если используется Docker (DB_HOST=postgres), даем время на инициализацию
	if cfg.Database.Host == "postgres" {
		log.Println("Обнаружен Docker режим (DB_HOST=postgres)")
		log.Println("Убедитесь, что PostgreSQL запущен: docker-compose up -d postgres")
		log.Println("Ожидание 2 секунды перед подключением...")
		time.Sleep(2 * time.Second)
	}

	log.Printf("Параметры подключения:")
	log.Printf("  Host: %s", cfg.Database.Host)
	log.Printf("  Port: %s", cfg.Database.Port)
	log.Printf("  User: %s", cfg.Database.User)
	log.Printf("  Database: %s", cfg.Database.DBName)
	log.Printf("  SSL Mode: %s", cfg.Database.SSLMode)

	// Пытаемся подключиться к базе данных
	db, err := database.NewConnection(&cfg.Database, cfg.AppEnv)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к базе данных: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка закрытия подключения: %v", err)
		}
	}()

	// Проверяем подключение через Ping
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Ошибка проверки подключения (Ping): %v", err)
		os.Exit(1)
	}

	log.Println("✅ Подключение к базе данных успешно установлено!")
	log.Println("✅ Проверка Ping прошла успешно!")

	// Дополнительная проверка - выполняем простой запрос
	var result int
	if err := db.Raw("SELECT 1").Scan(&result).Error; err != nil {
		log.Fatalf("❌ Ошибка выполнения тестового запроса: %v", err)
		os.Exit(1)
	}

	if result == 1 {
		log.Println("✅ Тестовый запрос выполнен успешно!")
		fmt.Println("\n🎉 Все проверки пройдены! База данных готова к работе.")
	} else {
		log.Fatalf("❌ Неожиданный результат тестового запроса: %d", result)
		os.Exit(1)
	}
}
