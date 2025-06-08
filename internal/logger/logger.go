package logger

import (
	"fmt"
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

func (l *appLogger) Info(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *appLogger) Warn(format string, args ...any) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l *appLogger) Error(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}
