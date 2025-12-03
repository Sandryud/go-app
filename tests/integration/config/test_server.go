//go:build integration
// +build integration

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgconn"

	appcfg "workout-app/internal/config"
	"workout-app/internal/database"
	"workout-app/internal/server"
)

// NewTestRouter создает новый экземпляр gin.Engine для интеграционных тестов.
// Использует отдельную тестовую БД, если задана переменная окружения TEST_DB_NAME.
func NewTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	rootDir, err := findProjectRoot()
	if err != nil {
		t.Fatalf("find project root: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir to project root: %v", err)
	}

	cfg, err := appcfg.Load()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}

	// Если указано имя тестовой БД — переопределяем его в конфиге.
	if testDB := os.Getenv("TEST_DB_NAME"); testDB != "" {
		cfg.Database.DBName = testDB
	}

	db, err := database.NewConnection(&cfg.Database, cfg.AppEnv)
	if err != nil {
		t.Fatalf("db connection: %v", err)
	}

	// Применяем миграцию и очищаем данные перед каждым тестом.
	if err := migrateUsers(db); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	if err := clearUsers(db); err != nil {
		t.Fatalf("clear users: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	srv := server.NewServer(cfg, db)
	return srv.GetRouter()
}

// findProjectRoot находит корень проекта по файлу go.mod.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// migrateUsers применяет SQL-миграцию для таблицы users.
func migrateUsers(db *database.DB) error {
	sqlBytes, err := os.ReadFile("internal/database/migrations/001_create_users_table.sql")
	if err != nil {
		return err
	}
	if err := db.Exec(string(sqlBytes)).Error; err != nil {
		// В интеграционных тестах миграция может быть уже применена (триггер/индексы существуют).
		// Игнорируем дубликат триггера "update_users_updated_at".
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "42710" && strings.Contains(pgErr.Message, "update_users_updated_at") {
				return nil
			}
		}
		// Fallback по тексту ошибки, если err не приведён к *pgconn.PgError
		msg := err.Error()
		if strings.Contains(msg, "42710") && strings.Contains(msg, "update_users_updated_at") {
			return nil
		}
		return err
	}
	return nil
}

// clearUsers очищает таблицу users перед тестом.
func clearUsers(db *database.DB) error {
	return db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error
}


