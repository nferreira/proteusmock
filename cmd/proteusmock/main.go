package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sophialabs/proteusmock/internal/app"
)

func main() {
	cfg := app.DefaultConfig()
	flag.StringVar(&cfg.RootDir, "root", cfg.RootDir, "root directory for mock scenarios")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "HTTP server port")
	flag.IntVar(&cfg.TraceSize, "trace-size", cfg.TraceSize, "number of trace entries to keep")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "log level (debug, info, warn, error)")
	flag.StringVar(&cfg.DefaultEngine, "default-engine", cfg.DefaultEngine, "default template engine for all scenarios (expr, jinja2)")
	flag.Parse()

	a, err := app.New(cfg)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "failed to initialize: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	if err := a.Run(context.Background()); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}
