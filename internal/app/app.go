package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/filesystem"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/logging"
	"github.com/sophialabs/proteusmock/internal/infrastructure/wiring"
)

// App is the thin lifecycle manager that delegates dependency construction to wiring.Container.
type App struct {
	cfg        Config
	container  *wiring.Container
	httpServer *http.Server
}

// New constructs the application by creating a logger, wiring infrastructure
// components via the container, and setting up the HTTP server.
func New(cfg Config) (*App, error) {
	level := parseLogLevel(cfg.LogLevel)
	logger := logging.New(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	container, err := wiring.New(wiring.Params{
		RootDir:        cfg.RootDir,
		TraceSize:      cfg.TraceSize,
		RateLimiterTTL: cfg.RateLimiterTTL,
		Logger:         logger,
		DefaultEngine:  cfg.DefaultEngine,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wire infrastructure: %w", err)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      container.Server(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &App{
		cfg:        cfg,
		container:  container,
		httpServer: httpServer,
	}, nil
}

// Run executes the full application lifecycle: load scenarios, start watcher,
// serve HTTP, and handle graceful shutdown on SIGINT/SIGTERM or context cancellation.
func (a *App) Run(ctx context.Context) error {
	defer a.container.Close()

	logger := a.container.Logger()
	server := a.container.Server()
	loadUC := a.container.LoadScenariosUseCase()

	idx, err := loadUC.Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}
	server.Rebuild(idx)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	watcher := a.setupWatcher()
	if watcher != nil {
		defer watcher.Stop()
	}
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting ProteusMock server", "addr", a.httpServer.Addr, "root", a.cfg.RootDir)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	logger.Info("server stopped")
	return nil
}

func (a *App) setupWatcher() *filesystem.Watcher {
	logger := a.container.Logger()
	server := a.container.Server()
	loadUC := a.container.LoadScenariosUseCase()

	watcher, err := filesystem.NewWatcher(a.cfg.RootDir, a.cfg.WatcherDebounce, logger, func() {
		newIdx, err := loadUC.Execute(context.Background())
		if err != nil {
			logger.Error("hot reload failed", "error", err)
			return
		}
		server.Rebuild(newIdx)
		logger.Info("hot reload complete")
	})
	if err != nil {
		logger.Warn("file watcher not available", "error", err)
		return nil
	}

	watcher.Start()
	logger.Info("file watcher started", "root", a.cfg.RootDir)
	return watcher
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
