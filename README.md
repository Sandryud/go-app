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

### Миграции базы данных

Проект использует библиотеку [golang-migrate](https://github.com/golang-migrate/migrate) для управления миграциями базы данных. Все SQL файлы миграций встроены в бинарник через `go:embed`.

#### Применение миграций

```bash
# Применить все доступные миграции (по умолчанию)
make migrate-up

# Или напрямую через команду
go run ./cmd/migrate
go run ./cmd/migrate -up
```

#### Откат миграций

```bash
# Откатить последнюю примененную миграцию
make migrate-down
go run ./cmd/migrate -down
```

#### Применение/откат N миграций

```bash
# Применить 2 миграции
make migrate-steps STEPS=2
go run ./cmd/migrate -steps 2

# Откатить 1 миграцию
make migrate-steps STEPS=-1
go run ./cmd/migrate -steps -1
```

#### Проверка версии

```bash
# Показать текущую версию миграций
make migrate-version
go run ./cmd/migrate -version
```

#### Справка по командам

```bash
go run ./cmd/migrate -help
```

#### Формат файлов миграций

Миграции должны следовать формату golang-migrate:
- `{version}_{name}.up.sql` - применение миграции
- `{version}_{name}.down.sql` - откат миграции

Пример: `000001_create_users_table.up.sql`, `000001_create_users_table.down.sql`

Все миграции находятся в `internal/database/migrations/` и автоматически встраиваются в бинарник.

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

#### Основные команды

- `make run` - Запустить приложение локально
- `make build` - Собрать бинарник
- `make test` - Запустить тесты
- `make check-db` - Проверить подключение к БД

#### Команды миграций

- `make migrate-up` - Применить все миграции
- `make migrate-down` - Откатить последнюю миграцию
- `make migrate-version` - Показать текущую версию миграций
- `make migrate-steps STEPS=N` - Применить/откатить N миграций

#### Docker команды

- `make docker-up` - Запустить PostgreSQL в Docker
- `make docker-down` - Остановить Docker контейнеры
- `make docker-build` - Собрать Docker образ приложения
- `make docker-logs` - Показать логи Docker контейнеров

