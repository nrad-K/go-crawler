package logger

import (
	"log/slog"
)

type AppLogger interface {
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

type appLogger struct {
	logger *slog.Logger
}

func NewAppLogger(logger *slog.Logger) AppLogger {
	return &appLogger{
		logger: logger,
	}
}

func (l *appLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *appLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *appLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}
