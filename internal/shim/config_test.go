package shim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromEnvAndParsingHelpers(t *testing.T) {
	root := t.TempDir()
	goBin := filepath.Join(root, "hugo-mcp-go")
	if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write go bin stub: %v", err)
	}
	workDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	t.Setenv("HUGO_MCP_SHIM_BIND_ADDR", "127.0.0.1")
	t.Setenv("HUGO_MCP_SHIM_BIND_PORT", "18180")
	t.Setenv("HUGO_MCP_SHIM_BACKEND_TOKEN", "REDACTED")
	t.Setenv("HUGO_MCP_GO_BIN", goBin)
	t.Setenv("HUGO_MCP_GO_WORKDIR", workDir)
	t.Setenv("HUGO_MCP_CHILD_PATH", "/tmp/fakebin")
	t.Setenv("HUGO_MCP_LOG_LEVEL", "debug")
	t.Setenv("HUGO_MCP_REQUEST_TIMEOUT_MS", "1234")
	t.Setenv("HUGO_MCP_STARTUP_TIMEOUT_MS", "4321")
	t.Setenv("HUGO_MCP_MAX_REQUEST_BYTES", "2048")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.RequestTimeoutMS != 1234 || cfg.StartupTimeoutMS != 4321 || cfg.MaxRequestBytes != 2048 {
		t.Fatalf("unexpected parsed config: %#v", cfg)
	}
	if cfg.ChildPath != "/tmp/fakebin" {
		t.Fatalf("unexpected child path: %#v", cfg.ChildPath)
	}

	if got, err := parseOptionalInt("MISSING_INT", 99); err != nil || got != 99 {
		t.Fatalf("parseOptionalInt fallback = %d err=%v", got, err)
	}
	t.Setenv("BAD_INT", "nope")
	if _, err := parseOptionalInt("BAD_INT", 0); err == nil {
		t.Fatal("parseOptionalInt expected error")
	}
	if got, err := parseOptionalInt64("MISSING_INT64", 99); err != nil || got != 99 {
		t.Fatalf("parseOptionalInt64 fallback = %d err=%v", got, err)
	}
	t.Setenv("BAD_INT64", "nope")
	if _, err := parseOptionalInt64("BAD_INT64", 0); err == nil {
		t.Fatal("parseOptionalInt64 expected error")
	}

	if err := (Config{}).Validate(); err == nil {
		t.Fatal("Validate() expected error")
	}
}

func TestLoadConfigFromEnvFailuresAndValidateBranches(t *testing.T) {
	root := t.TempDir()
	goBin := filepath.Join(root, "hugo-mcp-go")
	if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write go bin stub: %v", err)
	}
	workDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}

	t.Setenv("HUGO_MCP_SHIM_BIND_ADDR", "127.0.0.1")
	t.Setenv("HUGO_MCP_SHIM_BACKEND_TOKEN", "REDACTED")
	t.Setenv("HUGO_MCP_GO_BIN", goBin)
	t.Setenv("HUGO_MCP_GO_WORKDIR", workDir)

	t.Setenv("HUGO_MCP_SHIM_BIND_PORT", "bad")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("LoadConfigFromEnv() expected bind port error")
	}

	t.Setenv("HUGO_MCP_SHIM_BIND_PORT", "18180")
	t.Setenv("HUGO_MCP_REQUEST_TIMEOUT_MS", "bad")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("LoadConfigFromEnv() expected request timeout error")
	}

	t.Setenv("HUGO_MCP_REQUEST_TIMEOUT_MS", "100")
	t.Setenv("HUGO_MCP_STARTUP_TIMEOUT_MS", "bad")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("LoadConfigFromEnv() expected startup timeout error")
	}

	t.Setenv("HUGO_MCP_STARTUP_TIMEOUT_MS", "100")
	t.Setenv("HUGO_MCP_MAX_REQUEST_BYTES", "bad")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("LoadConfigFromEnv() expected max request bytes error")
	}

	if err := (Config{
		BindAddr:         "127.0.0.1",
		BindPort:         18180,
		BackendToken:     "x",
		GoBin:            goBin,
		GoWorkDir:        workDir,
		RequestTimeoutMS: 1,
		StartupTimeoutMS: 1,
		MaxRequestBytes:  1,
		LogLevel:         "info",
	}).Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	badCfg := Config{
		BindAddr:         "127.0.0.1",
		BindPort:         18180,
		BackendToken:     "x",
		GoBin:            filepath.Join(root, "missing"),
		GoWorkDir:        workDir,
		RequestTimeoutMS: 1,
		StartupTimeoutMS: 1,
		MaxRequestBytes:  1,
		LogLevel:         "info",
	}
	if err := badCfg.Validate(); err == nil {
		t.Fatal("Validate() expected GoBin error")
	}

	badCfg.GoBin = goBin
	badCfg.GoWorkDir = filepath.Join(root, "missing")
	if err := badCfg.Validate(); err == nil {
		t.Fatal("Validate() expected workdir error")
	}

	badCfg.GoWorkDir = workDir
	badCfg.RequestTimeoutMS = 0
	if err := badCfg.Validate(); err == nil {
		t.Fatal("Validate() expected request timeout error")
	}
	badCfg.RequestTimeoutMS = 1
	badCfg.StartupTimeoutMS = 0
	if err := badCfg.Validate(); err == nil {
		t.Fatal("Validate() expected startup timeout error")
	}
	badCfg.StartupTimeoutMS = 1
	badCfg.MaxRequestBytes = 0
	if err := badCfg.Validate(); err == nil {
		t.Fatal("Validate() expected max request bytes error")
	}
	badCfg.MaxRequestBytes = 1
	badCfg.LogLevel = ""
	if err := badCfg.Validate(); err == nil {
		t.Fatal("Validate() expected log level error")
	}
}
