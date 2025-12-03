# Workout App

Backend fitness application built with Go using Clean Architecture.

## Project Structure

```
go-app/
├── cmd/
│   └── server/              # Application entry point
├── internal/
│   ├── config/              # Configuration
│   ├── domain/              # Business logic and entities
│   ├── repository/          # Data access layer
│   ├── usecase/             # Business logic layer (use cases)
│   ├── handler/             # HTTP handlers (controllers)
│   └── database/            # Database initialization and migrations
├── pkg/                     # Reusable packages
├── api/                     # API specification
├── migrations/              # SQL migrations
├── scripts/                 # Helper scripts
└── tests/                   # Tests
```

## Architecture

This project follows Clean Architecture principles with layered structure:

- **Domain Layer**: Core business entities and logic
- **Repository Layer**: Data access abstraction
- **UseCase Layer**: Application business logic
- **Handler Layer**: HTTP request/response handling

## Technology Stack

- **Framework**: Gin
- **Database**: PostgreSQL
- **ORM**: GORM
- **Architecture**: Clean Architecture (Layered)

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL 12+

### Installation

1. Clone the repository
2. Copy `.env.example` to `.env` and configure
3. Run migrations
4. Start the server

### Docker Setup

Для запуска PostgreSQL через Docker:

```bash
# Запустить PostgreSQL
make docker-up

# Проверить подключение к базе данных
make check-db

# Остановить контейнеры
make docker-down

# Просмотр логов
make docker-logs
```

Или используйте docker-compose напрямую:

```bash
# Запустить PostgreSQL
docker-compose up -d postgres

# Проверить статус
docker-compose ps

# Остановить
docker-compose down
```

### Проверка подключения к базе данных

Перед запуском сервера рекомендуется проверить подключение к базе данных:

```bash
make check-db
```

Или напрямую:

```bash
go run scripts/check-db.go
```

Скрипт проверит:
- Загрузку конфигурации
- Подключение к базе данных
- Выполнение Ping
- Выполнение тестового SQL запроса

## Development

### Запуск приложения

```bash
# Локально (требуется запущенный PostgreSQL)
make run

# Или через Docker Compose (запускает и PostgreSQL, и приложение)
docker-compose up
```

### Доступные команды

- `make run` - Запустить приложение локально
- `make build` - Собрать бинарник
- `make test` - Запустить тесты
- `make check-db` - Проверить подключение к БД
- `make docker-up` - Запустить PostgreSQL в Docker
- `make docker-down` - Остановить Docker контейнеры
- `make docker-build` - Собрать Docker образ приложения

