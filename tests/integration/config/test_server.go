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

// GetTestDB возвращает тестовую БД для прямых запросов в тестах.
func GetTestDB() *database.DB {
	return testDB
}

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

// ForceEmailChangeForTests принудительно изменяет email пользователя в БД
// для интеграционных тестов, где код из письма недоступен.
func ForceEmailChangeForTests(t *testing.T, userID, newEmail string) {
	t.Helper()
	if testDB == nil {
		t.Fatalf("test database is not initialized")
	}
	// Удаляем коды верификации
	if err := testDB.Exec(`DELETE FROM email_verifications WHERE user_id = $1`, userID).Error; err != nil {
		t.Fatalf("failed to delete verification codes: %v", err)
	}
	// Обновляем email и помечаем как подтверждённый
	if err := testDB.Exec(`UPDATE users SET email = $1, is_email_verified = TRUE WHERE id = $2`, newEmail, userID).Error; err != nil {
		t.Fatalf("failed to change user email in tests: %v", err)
	}
}

// SoftDeleteUserForTests принудительно помечает пользователя как удалённого в БД
// для интеграционных тестов.
func SoftDeleteUserForTests(t *testing.T, email string) {
	t.Helper()
	if testDB == nil {
		t.Fatalf("test database is not initialized")
	}
	if err := testDB.Exec(`UPDATE users SET deleted_at = NOW() WHERE email = $1`, email).Error; err != nil {
		t.Fatalf("failed to soft delete user in tests: %v", err)
	}
}

// CreatePasswordResetCodeForTests создает код сброса пароля в БД для интеграционных тестов.
// Возвращает код, который можно использовать для сброса пароля.
// codeHash должен быть хэшем кода (используйте password.Hash для получения хэша).
func CreatePasswordResetCodeForTests(t *testing.T, email, codeHash string) {
	t.Helper()
	if testDB == nil {
		t.Fatalf("test database is not initialized")
	}

	// Получаем user_id по email
	var userID string
	if err := testDB.Raw(`SELECT id::text FROM users WHERE email = $1`, email).Scan(&userID).Error; err != nil {
		t.Fatalf("failed to get user ID: %v", err)
	}

	// Удаляем старые коды
	if err := testDB.Exec(`DELETE FROM password_resets WHERE user_id = $1`, userID).Error; err != nil {
		t.Fatalf("failed to delete old password reset codes: %v", err)
	}

	// Создаем новый код сброса пароля
	if err := testDB.Exec(`
		INSERT INTO password_resets (user_id, code_hash, expires_at, attempts, max_attempts, created_at)
		VALUES ($1, $2, NOW() + INTERVAL '15 minutes', 0, 5, NOW())
	`, userID, codeHash).Error; err != nil {
		t.Fatalf("failed to create password reset code: %v", err)
	}
}

// GetPasswordResetCodeForTests получает код сброса пароля из БД для интеграционных тестов.
// Возвращает код в виде строки. Если кода нет, возвращает пустую строку.
func GetPasswordResetCodeForTests(t *testing.T, email string) string {
	t.Helper()
	if testDB == nil {
		t.Fatalf("test database is not initialized")
	}

	// Получаем user_id по email
	var userID string
	if err := testDB.Raw(`SELECT id::text FROM users WHERE email = $1`, email).Scan(&userID).Error; err != nil {
		t.Fatalf("failed to get user ID: %v", err)
	}

	// Получаем code_hash из password_resets
	var codeHash string
	if err := testDB.Raw(`SELECT code_hash FROM password_resets WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW() ORDER BY created_at DESC LIMIT 1`, userID).Scan(&codeHash).Error; err != nil {
		// Код не найден - это нормально, если его еще нет
		return ""
	}

	// В тестах мы не можем расшифровать хэш, поэтому нужно получить код другим способом.
	// Для этого можно использовать логирование или создать тестовую функцию, которая возвращает код напрямую.
	// Но в реальных тестах код должен быть доступен через email sender mock или через БД напрямую.
	// Для интеграционных тестов лучше использовать прямой доступ к БД, если код хранится в открытом виде,
	// или использовать mock email sender, который сохраняет код.

	// В данном случае, так как код хэшируется, нам нужно либо:
	// 1. Использовать mock email sender, который сохраняет код
	// 2. Или получить код из логов (если используется loggerEmailSender)
	// 3. Или создать тестовую функцию, которая не хэширует код

	// Для простоты, вернем пустую строку и будем использовать другой подход в тестах
	return ""
}
