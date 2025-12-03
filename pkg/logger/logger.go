package logger

import (
	"log"
)

// Logger описывает минимальный интерфейс структурированного логгера,
// достаточный для использования в handler'ах и middleware.
type Logger interface {
	Info(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

type stdLogger struct{}

// Default возвращает простой логгер на базе стандартного log.Printf.
// В будущем реализацию можно заменить на zap/logrus/zerolog без изменения интерфейса.
func Default() Logger {
	return &stdLogger{}
}

func (l *stdLogger) Info(msg string, fields map[string]any) {
	log.Printf("INFO: %s %v", msg, fields)
}

func (l *stdLogger) Error(msg string, fields map[string]any) {
	log.Printf("ERROR: %s %v", msg, fields)
}


