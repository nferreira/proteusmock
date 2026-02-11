package app

import "time"

// Config holds all configurable parameters for the application.
type Config struct {
	RootDir   string
	Port      int
	TraceSize int
	LogLevel  string

	RateLimiterTTL  time.Duration
	WatcherDebounce time.Duration

	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	DefaultEngine string // "" = static, "expr", "jinja2"
}

// DefaultConfig returns a Config with sensible production defaults.
func DefaultConfig() Config {
	return Config{
		RootDir:   "./mock",
		Port:      8080,
		TraceSize: 200,
		LogLevel:  "debug",

		RateLimiterTTL:  10 * time.Minute,
		WatcherDebounce: 500 * time.Millisecond,

		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}
