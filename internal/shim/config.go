package shim

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultRequestTimeoutMS   = 30000
	DefaultStartupTimeoutMS   = 20000
	DefaultMaxRequestBytes    = 1 << 20
	DefaultLogLevel           = "info"
	DefaultRestartMaxDelayMS  = 5000
	DefaultRestartBaseDelayMS = 250
	DefaultRestartMaxRetries  = 5
)

type Config struct {
	BindAddr         string
	BindPort         int
	BackendToken     string
	GoBin            string
	GoWorkDir        string
	ChildPath        string
	RequestTimeoutMS int
	StartupTimeoutMS int
	MaxRequestBytes  int64
	LogLevel         string
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		BindAddr:         strings.TrimSpace(os.Getenv("HUGO_MCP_SHIM_BIND_ADDR")),
		BackendToken:     strings.TrimSpace(os.Getenv("HUGO_MCP_SHIM_BACKEND_TOKEN")),
		GoBin:            strings.TrimSpace(os.Getenv("HUGO_MCP_GO_BIN")),
		GoWorkDir:        strings.TrimSpace(os.Getenv("HUGO_MCP_GO_WORKDIR")),
		ChildPath:        strings.TrimSpace(os.Getenv("HUGO_MCP_CHILD_PATH")),
		LogLevel:         strings.TrimSpace(os.Getenv("HUGO_MCP_LOG_LEVEL")),
		RequestTimeoutMS: DefaultRequestTimeoutMS,
		StartupTimeoutMS: DefaultStartupTimeoutMS,
		MaxRequestBytes:  DefaultMaxRequestBytes,
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = DefaultLogLevel
	}

	var err error
	if cfg.BindPort, err = parseRequiredInt("HUGO_MCP_SHIM_BIND_PORT"); err != nil {
		return Config{}, err
	}
	if cfg.RequestTimeoutMS, err = parseOptionalInt("HUGO_MCP_REQUEST_TIMEOUT_MS", DefaultRequestTimeoutMS); err != nil {
		return Config{}, err
	}
	if cfg.StartupTimeoutMS, err = parseOptionalInt("HUGO_MCP_STARTUP_TIMEOUT_MS", DefaultStartupTimeoutMS); err != nil {
		return Config{}, err
	}
	if maxReq, err := parseOptionalInt64("HUGO_MCP_MAX_REQUEST_BYTES", DefaultMaxRequestBytes); err != nil {
		return Config{}, err
	} else {
		cfg.MaxRequestBytes = maxReq
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch {
	case c.BindAddr == "":
		return fmt.Errorf("missing required env HUGO_MCP_SHIM_BIND_ADDR")
	case c.BindPort <= 0:
		return fmt.Errorf("invalid HUGO_MCP_SHIM_BIND_PORT: must be > 0")
	case c.BackendToken == "":
		return fmt.Errorf("missing required env HUGO_MCP_SHIM_BACKEND_TOKEN")
	case c.GoBin == "":
		return fmt.Errorf("missing required env HUGO_MCP_GO_BIN")
	case c.GoWorkDir == "":
		return fmt.Errorf("missing required env HUGO_MCP_GO_WORKDIR")
	case c.RequestTimeoutMS <= 0:
		return fmt.Errorf("invalid HUGO_MCP_REQUEST_TIMEOUT_MS: must be > 0")
	case c.StartupTimeoutMS <= 0:
		return fmt.Errorf("invalid HUGO_MCP_STARTUP_TIMEOUT_MS: must be > 0")
	case c.MaxRequestBytes <= 0:
		return fmt.Errorf("invalid HUGO_MCP_MAX_REQUEST_BYTES: must be > 0")
	case c.LogLevel == "":
		return fmt.Errorf("missing log level")
	}
	if info, err := os.Stat(c.GoBin); err != nil {
		return fmt.Errorf("invalid HUGO_MCP_GO_BIN: %w", err)
	} else if info.IsDir() {
		return fmt.Errorf("invalid HUGO_MCP_GO_BIN: must be a file")
	}
	if info, err := os.Stat(c.GoWorkDir); err != nil {
		return fmt.Errorf("invalid HUGO_MCP_GO_WORKDIR: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("invalid HUGO_MCP_GO_WORKDIR: must be a directory")
	}
	return nil
}

func parseRequiredInt(name string) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, fmt.Errorf("missing required env %s", name)
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}

func parseOptionalInt(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}

func parseOptionalInt64(name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}
