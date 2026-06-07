package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSecretFileRejectsMissingEmptyAndWorldReadable(t *testing.T) {
	dir := t.TempDir()

	if _, err := LoadSecretFile(filepath.Join(dir, "missing"), dir); err == nil {
		t.Fatal("expected missing secret to fail")
	}

	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatalf("write empty secret: %v", err)
	}
	if _, err := LoadSecretFile(empty, dir); err == nil {
		t.Fatal("expected empty secret to fail")
	}

	worldReadable := filepath.Join(dir, "world-readable")
	if err := os.WriteFile(worldReadable, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write world-readable secret: %v", err)
	}
	if _, err := LoadSecretFile(worldReadable, dir); err == nil {
		t.Fatal("expected world-readable secret to fail")
	}
}

func TestLoadSecretFileRejectsPathOutsideAllowedDirectory(t *testing.T) {
	allowed := t.TempDir()
	outside := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside secret: %v", err)
	}
	if _, err := LoadSecretFile(outside, allowed); err == nil {
		t.Fatal("expected secret outside allowed directory to fail")
	}
}

func TestLoadConfigFromEnvDefaultsToDryRun(t *testing.T) {
	t.Setenv("HUGO_POST_BUILD_HOOKS_ENABLED", "")
	t.Setenv("HUGO_CLOUDFLARE_PURGE_ENABLED", "")
	t.Setenv("HUGO_GOOGLE_INDEXING_ENABLED", "")
	t.Setenv("HUGO_INDEXNOW_ENABLED", "")
	t.Setenv("HUGO_HOOKS_DRY_RUN", "")
	t.Setenv("HUGO_HOOKS_MAX_RETRIES", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if !cfg.HooksDryRun {
		t.Fatal("expected dry-run to default to true")
	}
	if cfg.HooksMaxRetries != 5 {
		t.Fatalf("expected max retries default 5, got %d", cfg.HooksMaxRetries)
	}
	if cfg.CloudflarePurgeEnabled || cfg.GoogleIndexingEnabled || cfg.IndexNowEnabled {
		t.Fatal("expected provider integrations to default to disabled")
	}
}
