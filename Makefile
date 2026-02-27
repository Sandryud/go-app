
help: ## Показать это сообщение с помощью
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Запустить приложение
	@go run cmd/server/main.go

build: ## Собрать приложение
	@go build -o bin/server cmd/server/main.go

test: ## Запустить тесты
	@go test -v ./...

test-coverage: ## Запустить тесты с покрытием
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean: ## Очистить артефакты сборки
	@rm -rf bin/
	@rm -f coverage.out coverage.html

migrate-up: ## Применить все миграции базы данных
	@echo "Применение миграций базы данных..."
	@go run ./cmd/migrate -up

migrate-down: ## Откатить последнюю миграцию базы данных
	@echo "Откат последней миграции..."
	@go run ./cmd/migrate -down

migrate-version: ## Показать текущую версию миграций
	@echo "Текущая версия миграций:"
	@go run ./cmd/migrate -version

migrate-steps: ## Применить/откатить N миграций (использование: make migrate-steps STEPS=2)
	@if [ -z "$(STEPS)" ]; then \
		echo "Ошибка: укажите количество шагов через переменную STEPS"; \
		echo "Пример: make migrate-steps STEPS=2 (применить 2 миграции)"; \
		echo "Пример: make migrate-steps STEPS=-1 (откатить 1 миграцию)"; \
		exit 1; \
	fi
	@echo "Применение $(STEPS) миграций..."
	@go run ./cmd/migrate -steps $(STEPS)

tidy: ## Очистить go модули
	@go mod tidy

vet: ## Запустить go vet
	@go vet ./...

fmt: ## Форматировать код
	@go fmt ./...

test-integration: ## Запустить интеграционные тесты (используется TEST_DB_NAME=workout_app_test)
	@echo "Запуск интеграционных тестов (TEST_DB_NAME=workout_app_test)..."
	@export TEST_DB_NAME=workout_app_test; \
	go test ./tests/integration/... -tags=integration

docker-up: ## Запустить Docker Compose (PostgreSQL)
	@docker-compose up -d postgres
	@echo "Ожидание готовности PostgreSQL..."
	@sleep 5
	@docker-compose ps

docker-down: ## Остановить Docker Compose
	@docker-compose down

docker-logs: ## Показать логи Docker Compose
	@docker-compose logs -f

docker-build: ## Собрать Docker образ приложения
	@docker-compose build app

swagger: ## Сгенерировать Swagger/OpenAPI документацию
	@echo "Генерация Swagger документации..."
	@SWAG="$$(command -v swag 2>/dev/null || echo "$$(go env GOPATH)/bin/swag")"; \
	if [ -z "$$SWAG" ] || [ ! -x "$$SWAG" ]; then \
		echo "Ошибка: swag не найден. Установите его командой: go install github.com/swaggo/swag/cmd/swag@latest"; \
		echo "Убедитесь, что $$(go env GOPATH)/bin в PATH"; \
		exit 1; \
	fi; \
	"$$SWAG" init -g cmd/server/main.go -o api/swagger --parseDependency --parseInternal
	@echo "Swagger документация успешно сгенерирована в api/swagger/"

exercises-json: ## Сгенерировать exercises.json из CSV (data/csv → dist/exercises.json)
	@echo "Генерация каталога упражнений из CSV..."
	@go run ./cmd/csv2json -csv data/csv -out dist/exercises.json
	@echo "Каталог записан в dist/exercises.json"

validate-csv: ## Проверить CSV-файлы упражнений в data/csv
	@echo "Валидация CSV в data/csv..."
	@go run ./cmd/csv-validate -csv data/csv

