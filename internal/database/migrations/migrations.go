package migrations

import "embed"

// Migrations содержит все SQL файлы миграций, встроенные в бинарник.
// Используется для загрузки миграций через golang-migrate.
//
//go:embed *.sql
var Migrations embed.FS
