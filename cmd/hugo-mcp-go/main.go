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

	cfg, err := config.LoadFromEnv()
	if err != nil {
		return err
	}

	svc, err := server.New(cfg)
	if err != nil {
		return err
	}
	return svc.RunStdio(ctx)
}
