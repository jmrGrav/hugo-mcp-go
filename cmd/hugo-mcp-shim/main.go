package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
	"github.com/jmrGrav/hugo-mcp-go/internal/shim"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hugo-mcp-shim: %s\n", observability.RedactString(err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := shim.LoadConfigFromEnv()
	if err != nil {
		return err
	}
	srv, err := shim.NewServer(cfg, nil)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = srv.Close(context.Background())
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
