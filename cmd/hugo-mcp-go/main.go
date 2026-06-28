package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmrGrav/hugo-mcp-go/internal/config"
	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
	"github.com/jmrGrav/hugo-mcp-go/internal/server"
	"github.com/jmrGrav/hugo-mcp-go/internal/transport"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hugo-mcp-go: %s\n", observability.RedactString(err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := observability.New()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		return err
	}
	logger.Info("hugo-mcp-go starting",
		"hugo_root", cfg.HugoRoot,
		"content_root", cfg.ContentRoot,
		"static_root", cfg.StaticRoot,
		"public_root", cfg.HugoRoot+"/public",
		"work_root", cfg.HugoRoot+"/work",
	)

	transportCfg, err := transport.LoadFromEnv()
	if err != nil {
		return err
	}

	svc, err := server.New(cfg)
	if err != nil {
		return err
	}
	if transportCfg.Transport == "http" {
		return svc.RunHTTP(ctx, transportCfg, logger)
	}
	return svc.RunStdio(ctx)
}
