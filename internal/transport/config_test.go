package transport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnvDefaultsToStdio(t *testing.T) {
	t.Setenv("HUGO_MCP_TRANSPORT", "")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.Transport != "stdio" {
		t.Fatalf("Transport = %q want stdio", cfg.Transport)
	}
}

func TestLoadFromEnvHTTPRequiresToken(t *testing.T) {
	t.Setenv("HUGO_MCP_TRANSPORT", "http")
	t.Setenv("HUGO_MCP_HTTP_BIND_PORT", "18181")
	t.Setenv("HUGO_MCP_HTTP_BIND_ADDR", "127.0.0.1")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("LoadFromEnv() expected error")
	}
}

func TestLoadFromEnvHTTPTokenFile(t *testing.T) {
	root := t.TempDir()
	tokenFile := filepath.Join(root, "token.txt")
	if err := os.WriteFile(tokenFile, []byte("local-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	t.Setenv("HUGO_MCP_TRANSPORT", "http")
	t.Setenv("HUGO_MCP_HTTP_BIND_PORT", "18181")
	t.Setenv("HUGO_MCP_HTTP_TOKEN_FILE", tokenFile)

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.HTTPToken != "local-token" {
		t.Fatalf("HTTPToken = %q want local-token", cfg.HTTPToken)
	}
}

func TestLoadFromEnvHTTPDefaultsLoopbackBind(t *testing.T) {
	root := t.TempDir()
	tokenFile := filepath.Join(root, "token.txt")
	if err := os.WriteFile(tokenFile, []byte("local-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	t.Setenv("HUGO_MCP_TRANSPORT", "http")
	t.Setenv("HUGO_MCP_HTTP_BIND_PORT", "18181")
	t.Setenv("HUGO_MCP_HTTP_TOKEN_FILE", tokenFile)

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.HTTPBindAddr != "127.0.0.1" {
		t.Fatalf("HTTPBindAddr = %q want 127.0.0.1", cfg.HTTPBindAddr)
	}
}
