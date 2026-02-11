package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/logging"
)

func TestSlogLogger_AllLevels(t *testing.T) {
	tests := []struct {
		name  string
		call  func(l *logging.SlogLogger)
		level string
	}{
		{"Info", func(l *logging.SlogLogger) { l.Info("info message", "key", "val") }, "INFO"},
		{"Warn", func(l *logging.SlogLogger) { l.Warn("warn message", "key", "val") }, "WARN"},
		{"Error", func(l *logging.SlogLogger) { l.Error("error message", "key", "val") }, "ERROR"},
		{"Debug", func(l *logging.SlogLogger) { l.Debug("debug message", "key", "val") }, "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			logger := logging.New(slog.New(handler))

			tt.call(logger)

			output := buf.String()
			if !strings.Contains(output, tt.level) {
				t.Errorf("expected output to contain %q, got: %s", tt.level, output)
			}
			if !strings.Contains(output, "key=val") {
				t.Errorf("expected output to contain key=val, got: %s", output)
			}
		})
	}
}
