package logging

import (
	"log/slog"

	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
)

var _ ports.Logger = (*SlogLogger)(nil)

// SlogLogger wraps slog to implement ports.Logger.
type SlogLogger struct {
	logger *slog.Logger
}

// New creates a new SlogLogger from an slog.Logger.
func New(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{logger: logger}
}

func (l *SlogLogger) Info(msg string, args ...any)  { l.logger.Info(msg, args...) }
func (l *SlogLogger) Warn(msg string, args ...any)  { l.logger.Warn(msg, args...) }
func (l *SlogLogger) Error(msg string, args ...any) { l.logger.Error(msg, args...) }
func (l *SlogLogger) Debug(msg string, args ...any) { l.logger.Debug(msg, args...) }
