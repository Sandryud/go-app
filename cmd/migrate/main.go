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
		force      = flag.String("force", "", "Принудительно установить версию без выполнения миграций (например 0), для снятия dirty")
		fixDirty0  = flag.Bool("fix-dirty-0", false, "SQL: снять dirty при version=0 (обход ошибки «read down for version 0»), затем запустите -up")
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
		fmt.Fprintf(os.Stderr, "  %s -force 0     # Снять dirty, установить версию 0 (после ошибки миграции)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -fix-dirty-0 # SQL: снять dirty при version=0, затем -up\n", os.Args[0])
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
	if *force != "" {
		actionCount++
	}
	if *fixDirty0 {
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
	case *force != "":
		handleForce(migrator, *force)
	case *fixDirty0:
		handleFixDirty0(db)
	case *up:
		handleUp(migrator)
	}
}

// handleUp применяет все доступные миграции
func handleUp(migrator *database.Migrator) {
	dirty, err := migrator.CheckDirty()
	if err != nil {
		log.Fatalf("Ошибка проверки состояния миграций: %v", err)
	}
	if dirty {
		ver, _, _ := migrator.Version()
		log.Fatalf(
			"База в состоянии dirty (версия %d). Сначала выполните: %s -force %d , затем снова -up.",
			ver, os.Args[0], ver,
		)
	}
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

// handleFixDirty0 выполняет SQL: удаляет запись с version=0 из schema_migrations (обход ошибки «read down for version 0»).
// Пустая таблица трактуется как «миграции не применялись», тогда -up применит все миграции с 1.
func handleFixDirty0(db *database.DB) {
	log.Println("Очистка version=0 в schema_migrations (DELETE)...")
	res := db.Exec("DELETE FROM schema_migrations WHERE version = 0")
	if res.Error != nil {
		log.Fatalf("Ошибка очистки: %v", res.Error)
	}
	if res.RowsAffected == 0 {
		log.Println("Затронуто 0 строк (таблица уже без version=0). Запустите -up.")
	} else {
		log.Printf("Удалено строк: %d. Запустите -up для применения миграций.", res.RowsAffected)
	}
}

// handleForce принудительно устанавливает версию миграции без выполнения миграций.
// Используется для восстановления после dirty (например: -force 0, затем -up).
func handleForce(migrator *database.Migrator, forceStr string) {
	version, err := strconv.Atoi(forceStr)
	if err != nil {
		log.Fatalf("Ошибка: неверный формат версии для -force (ожидается число): %v", err)
	}
	log.Printf("Принудительная установка версии миграции на %d...", version)
	if err := migrator.Force(version); err != nil {
		log.Fatalf("Ошибка принудительной установки версии: %v", err)
	}
	log.Println("Версия установлена. Запустите -up для применения миграций.")
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
