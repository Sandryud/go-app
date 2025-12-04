//go:build integration
// +build integration

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	appcfg "workout-app/internal/config"
	"workout-app/internal/database"
	"workout-app/internal/server"
)

var testDB *database.DB

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

	testDB = db

	// Применяем миграции и очищаем данные перед каждым тестом.
	if err := MigrateDatabase(db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := clearUsers(db); err != nil {
		t.Fatalf("clear users: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		testDB = nil
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

// clearUsers очищает таблицу users перед тестом.
func clearUsers(db *database.DB) error {
	return db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error
}

// VerifyUserEmailForTests принудительно помечает email как подтверждённый в БД
// для интеграционных сценариев, где код из письма недоступен.
func VerifyUserEmailForTests(t *testing.T, email string) {
	t.Helper()
	if testDB == nil {
		t.Fatalf("test database is not initialized")
	}
	if err := testDB.Exec(`UPDATE users SET is_email_verified = TRUE WHERE email = $1`, email).Error; err != nil {
		t.Fatalf("failed to verify user email in tests: %v", err)
	}
}
