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

## Development

To be implemented...

