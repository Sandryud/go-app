.PHONY: help run build test clean migrate-up migrate-down

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

migrate-up: ## Применить миграции базы данных
	@echo "TODO: Реализовать применение миграций"

migrate-down: ## Откатить миграции базы данных
	@echo "TODO: Реализовать откат миграций"

tidy: ## Очистить go модули
	@go mod tidy

vet: ## Запустить go vet
	@go vet ./...

fmt: ## Форматировать код
	@go fmt ./...

