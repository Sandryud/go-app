package main

import (
	"log"

	"workout-app/internal/config"
)

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

	// TODO: Инициализировать сервер с конфигурацией
}
