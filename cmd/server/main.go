package main

import (
	"log"

	"workout-app/internal/config"
	"workout-app/internal/database"
	"workout-app/internal/server"
)

// @title           Workout App API
// @version         1.0
// @description     API фитнес-приложения: аутентификация (JWT access + refresh) и профиль пользователя.
// @BasePath        /
//
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                В формате: "Bearer <access_token>"
func main() {
	log.Println("Workout App Server Starting...")

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	log.Printf("Конфигурация загружена успешно")
	log.Printf("Сервер будет запущен на %s", cfg.Server.Address())
	log.Printf("База данных: %s@%s:%s/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	log.Printf("JWT: issuer=%s access_ttl=%s refresh_ttl=%s", cfg.JWT.Issuer, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

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

	// Создаем и запускаем HTTP сервер
	srv := server.NewServer(cfg, db)
	if err := srv.Start(); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
