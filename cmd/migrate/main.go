package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"workout-app/internal/config"
	"workout-app/internal/database"
)

func main() {
	// Определяем флаги
	var (
		up      = flag.Bool("up", false, "Применить все доступные миграции (по умолчанию)")
		down    = flag.Bool("down", false, "Откатить последнюю миграцию")
		steps   = flag.String("steps", "", "Применить/откатить N миграций (положительное число - вверх, отрицательное - вниз)")
		version = flag.Bool("version", false, "Показать текущую версию миграции")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [опции]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Опции:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nПримеры:\n")
		fmt.Fprintf(os.Stderr, "  %s              # Применить все миграции (по умолчанию)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -up          # Применить все миграции\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -down        # Откатить последнюю миграцию\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -steps 2     # Применить 2 миграции\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -steps -1    # Откатить 1 миграцию\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version     # Показать текущую версию\n", os.Args[0])
	}

	flag.Parse()

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

	// Создаем мигратор
	migrator, err := database.NewMigrator(db)
	if err != nil {
		log.Fatalf("Ошибка создания мигратора: %v", err)
	}
	defer func() {
		if err := migrator.Close(); err != nil {
			log.Printf("Ошибка закрытия мигратора: %v", err)
		}
	}()

	// Определяем действие на основе флагов
	actionCount := 0
	if *up {
		actionCount++
	}
	if *down {
		actionCount++
	}
	if *steps != "" {
		actionCount++
	}
	if *version {
		actionCount++
	}

	// Если не указано действие, по умолчанию применяем все миграции
	if actionCount == 0 {
		*up = true
	} else if actionCount > 1 {
		log.Fatal("Ошибка: можно указать только одно действие за раз")
	}

	// Выполняем действие
	switch {
	case *version:
		handleVersion(migrator)
	case *down:
		handleDown(migrator)
	case *steps != "":
		handleSteps(migrator, *steps)
	case *up:
		handleUp(migrator)
	}
}

// handleUp применяет все доступные миграции
func handleUp(migrator *database.Migrator) {
	log.Println("Применение всех доступных миграций...")
	if err := migrator.Up(); err != nil {
		if err == database.ErrNoChange {
			log.Println("Нет миграций для применения. База данных уже актуальна.")
			return
		}
		log.Fatalf("Ошибка применения миграций: %v", err)
	}
	log.Println("Все миграции успешно применены")
}

// handleDown откатывает последнюю миграцию
func handleDown(migrator *database.Migrator) {
	log.Println("Откат последней миграции...")
	if err := migrator.Down(); err != nil {
		if err == database.ErrNoChange {
			log.Println("Нет миграций для отката. База данных уже в базовом состоянии.")
			return
		}
		log.Fatalf("Ошибка отката миграции: %v", err)
	}
	log.Println("Миграция успешно откатилась")
}

// handleSteps применяет или откатывает N миграций
func handleSteps(migrator *database.Migrator, stepsStr string) {
	n, err := strconv.Atoi(stepsStr)
	if err != nil {
		log.Fatalf("Ошибка: неверный формат числа для -steps: %v", err)
	}

	if n == 0 {
		log.Println("Ноль миграций для применения/отката")
		return
	}

	direction := "вверх"
	absN := n
	if n < 0 {
		direction = "вниз"
		absN = -n
	}

	log.Printf("Применение %d миграций %s...\n", absN, direction)

	if err := migrator.Steps(n); err != nil {
		if err == database.ErrNoChange {
			log.Printf("Нет миграций для применения/отката в направлении %s.\n", direction)
			return
		}
		log.Fatalf("Ошибка применения миграций: %v", err)
	}
}

// handleVersion показывает текущую версию миграции
func handleVersion(migrator *database.Migrator) {
	version, dirty, err := migrator.Version()
	if err != nil {
		log.Fatalf("Ошибка получения версии: %v", err)
	}

	if version == 0 {
		log.Println("Версия: нет примененных миграций")
		return
	}

	if dirty {
		log.Printf("Версия: %d (ГРЯЗНОЕ СОСТОЯНИЕ - требуется ручное вмешательство!)\n", version)
		os.Exit(1)
	} else {
		log.Printf("Версия: %d\n", version)
	}
}
