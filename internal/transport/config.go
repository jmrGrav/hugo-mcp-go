package transport

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultMode             = "stdio"
	DefaultHTTPBindAddr     = "127.0.0.1"
	DefaultMaxRequestBytes  = 1 << 20
	DefaultMaxChunkBytes    = 256 << 10
	DefaultMaxResponseBytes = 1 << 20
)

type Config struct {
	Transport        string
	HTTPBindAddr     string
	HTTPBindPort     int
	HTTPToken        string
	HTTPTokenFile    string
	StreamingEnabled bool
	MaxRequestBytes  int64
	MaxChunkBytes    int64
	MaxResponseBytes int64
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		Transport:        strings.TrimSpace(os.Getenv("HUGO_MCP_TRANSPORT")),
		HTTPBindAddr:     strings.TrimSpace(os.Getenv("HUGO_MCP_HTTP_BIND_ADDR")),
		HTTPToken:        strings.TrimSpace(os.Getenv("HUGO_MCP_HTTP_TOKEN")),
		HTTPTokenFile:    strings.TrimSpace(os.Getenv("HUGO_MCP_HTTP_TOKEN_FILE")),
		StreamingEnabled: boolEnv("HUGO_MCP_STREAMING_ENABLED"),
		MaxRequestBytes:  int64Env("HUGO_MCP_MAX_REQUEST_BYTES", DefaultMaxRequestBytes),
		MaxChunkBytes:    int64Env("HUGO_MCP_MAX_CHUNK_BYTES", DefaultMaxChunkBytes),
		MaxResponseBytes: int64Env("HUGO_MCP_MAX_RESPONSE_BYTES", DefaultMaxResponseBytes),
	}
	if cfg.Transport == "" {
		cfg.Transport = DefaultMode
	}
	if cfg.HTTPBindAddr == "" {
		cfg.HTTPBindAddr = DefaultHTTPBindAddr
	}

	port, err := intEnv("HUGO_MCP_HTTP_BIND_PORT")
	if err != nil {
		return Config{}, err
	}
	cfg.HTTPBindPort = port

	if cfg.Transport == "stdio" {
		return cfg, nil
	}
	if cfg.Transport != "http" {
		return Config{}, fmt.Errorf("invalid HUGO_MCP_TRANSPORT: must be stdio or http")
	}
	if cfg.HTTPBindPort <= 0 {
		return Config{}, fmt.Errorf("missing required env HUGO_MCP_HTTP_BIND_PORT")
	}
	if cfg.HTTPToken == "" {
		if cfg.HTTPTokenFile != "" {
			raw, err := os.ReadFile(cfg.HTTPTokenFile)
			if err != nil {
				return Config{}, fmt.Errorf("invalid HUGO_MCP_HTTP_TOKEN_FILE: %w", err)
			}
			cfg.HTTPToken = strings.TrimSpace(string(raw))
		}
	}
	if cfg.HTTPToken == "" {
		return Config{}, fmt.Errorf("missing required env HUGO_MCP_HTTP_TOKEN or HUGO_MCP_HTTP_TOKEN_FILE")
	}
	if cfg.MaxChunkBytes <= 0 {
		return Config{}, fmt.Errorf("invalid HUGO_MCP_MAX_CHUNK_BYTES: must be > 0")
	}
	if cfg.MaxRequestBytes <= 0 {
		return Config{}, fmt.Errorf("invalid HUGO_MCP_MAX_REQUEST_BYTES: must be > 0")
	}
	if cfg.MaxResponseBytes <= 0 {
		return Config{}, fmt.Errorf("invalid HUGO_MCP_MAX_RESPONSE_BYTES: must be > 0")
	}
	return cfg, nil
}

func boolEnv(name string) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func intEnv(name string) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}

func int64Env(name string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}
