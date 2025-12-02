package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

// Init инициализирует глобальный логгер
func Init(level string, jsonFormat bool) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// Get возвращает логгер
func Get() *slog.Logger {
	if defaultLogger == nil {
		Init("info", false)
	}
	return defaultLogger
}

// With создает дочерний логгер с дополнительными атрибутами
func With(args ...any) *slog.Logger {
	return Get().With(args...)
}

// Debug логирует отладочное сообщение
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info логирует информационное сообщение
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn логирует предупреждение
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error логирует ошибку
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}
