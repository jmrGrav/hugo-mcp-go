package hooks

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestConfigAndHelperBranches(t *testing.T) {
	if got := parseBoolEnv("HUGO_TEST_BOOL_MISSING", true); !got {
		t.Fatal("expected missing bool env to use fallback")
	}
	t.Setenv("HUGO_TEST_BOOL_BAD", "not-a-bool")
	if got := parseBoolEnv("HUGO_TEST_BOOL_BAD", false); got {
		t.Fatal("expected invalid bool env to use fallback")
	}
	t.Setenv("HUGO_TEST_INT_BAD", "not-an-int")
	if got := parseIntEnv("HUGO_TEST_INT_BAD", 7); got != 7 {
		t.Fatalf("expected invalid int env fallback, got %d", got)
	}
	t.Setenv("HUGO_TEST_VALUE", "  hi  ")
	if got := envOrDefault("HUGO_TEST_VALUE", "fallback"); got != "hi" {
		t.Fatalf("expected trimmed env value, got %q", got)
	}
	if got := envOrDefault("HUGO_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected env fallback, got %q", got)
	}
	if got, err := ioReadAllLimit(strings.NewReader("abcdef"), 0); err != nil || len(got) == 0 {
		t.Fatal("expected default limit read to return bytes")
	}
	if retryDelay(1) != 0 {
		t.Fatal("expected first retry delay to be zero")
	}
	if retryDelay(3) != 20_000_000 {
		t.Fatalf("expected third retry delay to be 20ms, got %s", retryDelay(3))
	}
	limited, err := ioReadAllLimit(strings.NewReader("abcdef"), 3)
	if err != nil {
		t.Fatalf("ioReadAllLimit error = %v", err)
	}
	if string(limited) != "abc" {
		t.Fatalf("unexpected limited read %q", string(limited))
	}
	if got := redactSecrets("Bearer secret-a and secret-b", "secret-a", "secret-b"); strings.Contains(got, "secret-a") || strings.Contains(got, "secret-b") {
		t.Fatalf("expected both secrets to be redacted, got %q", got)
	}
}

func TestLoadConfigFromEnvOverrides(t *testing.T) {
	t.Setenv("HUGO_POST_BUILD_HOOKS_ENABLED", "false")
	t.Setenv("HUGO_CLOUDFLARE_PURGE_ENABLED", "true")
	t.Setenv("HUGO_CLOUDFLARE_ALLOW_PURGE_EVERYTHING", "true")
	t.Setenv("HUGO_GOOGLE_INDEXING_ENABLED", "true")
	t.Setenv("HUGO_INDEXNOW_ENABLED", "true")
	t.Setenv("HUGO_HOOKS_DRY_RUN", "false")
	t.Setenv("HUGO_HOOKS_MAX_RETRIES", "9")
	t.Setenv("HUGO_HOOKS_ADMIN_ENABLED", "true")
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.PostBuildHooksEnabled || !cfg.CloudflarePurgeEnabled || !cfg.CloudflareAllowPurgeEverything || !cfg.GoogleIndexingEnabled || !cfg.IndexNowEnabled || cfg.HooksDryRun || cfg.HooksMaxRetries != 9 || !cfg.HooksAdminEnabled {
		t.Fatalf("unexpected config overrides: %#v", cfg)
	}
}

func TestLoadSecretFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	allowed := t.TempDir()
	secret := writeSecretFile(t, "secret.txt", []byte("secret"))
	link := dir + "/secret-link"
	if err := symlink(secret, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	if _, err := LoadSecretFile(link, allowed); err == nil {
		t.Fatal("expected symlink secret to fail")
	}
}

func TestStoreListAndUpdateJobs(t *testing.T) {
	store, err := OpenStore(t.TempDir() + "/hooks.db")
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	ctx := context.Background()
	id1, err := store.Enqueue(ctx, HookJob{Provider: "cloudflare", Action: "URL_UPDATED", TargetURLs: []string{"https://example.com/a"}, Status: "failed", LastError: "Bearer secret"})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	id2, err := store.Enqueue(ctx, HookJob{Provider: "indexnow", Action: "URL_DELETED", TargetURLs: []string{"https://example.com/b"}, Status: "pending"})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if id1 == "" || id2 == "" {
		t.Fatal("expected ids from enqueue")
	}
	jobs, err := store.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].ID == "" {
		t.Fatal("expected job id to be populated")
	}
	if _, err := store.SetJobStatus(ctx, []string{id1}, "pending"); err != nil {
		t.Fatalf("SetJobStatus() error = %v", err)
	}
	jobs, err = store.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if jobs[0].Status != "pending" && jobs[1].Status != "pending" {
		t.Fatal("expected updated pending status")
	}
	var nilStore *Store
	if err := nilStore.Close(); err != nil {
		t.Fatalf("nil Close() error = %v", err)
	}
}

func TestStoreValidationBranches(t *testing.T) {
	if _, err := OpenStore(""); err == nil {
		t.Fatal("expected empty store path to fail")
	}
	store, err := OpenStore(t.TempDir() + "/hooks.db")
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	if _, err := store.Enqueue(context.Background(), HookJob{}); err == nil {
		t.Fatal("expected empty hook job to fail")
	}
	if _, err := store.SetJobStatus(context.Background(), nil, "pending"); err != nil {
		t.Fatalf("SetJobStatus(nil) error = %v", err)
	}
	if err := store.RecordAudit(context.Background(), AuditRecord{}); err == nil {
		t.Fatal("expected empty audit record to fail")
	}
	if _, err := store.ListJobs(context.Background()); err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
}

func TestProvidersRunAndNameInDryRunMode(t *testing.T) {
	cloudflare := NewCloudflareClient(Config{CloudflarePurgeEnabled: true, HooksDryRun: true}, nil)
	if got := cloudflare.Name(); got != "cloudflare" {
		t.Fatalf("unexpected cloudflare name %q", got)
	}
	if res, err := cloudflare.Run(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err != nil || res.Status != "dry_run" {
		t.Fatalf("cloudflare Run() = %#v, %v", res, err)
	}

	googleClient := NewGoogleIndexingClient(Config{GoogleIndexingEnabled: true, HooksDryRun: true}, nil)
	if got := googleClient.Name(); got != "google_indexing" {
		t.Fatalf("unexpected google name %q", got)
	}
	if res, err := googleClient.Run(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err != nil || res.Status != "dry_run" {
		t.Fatalf("google Run() = %#v, %v", res, err)
	}

	indexNow := NewIndexNowClient(Config{IndexNowEnabled: true, HooksDryRun: true}, nil)
	if got := indexNow.Name(); got != "indexnow" {
		t.Fatalf("unexpected indexnow name %q", got)
	}
	if res, err := indexNow.Run(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err != nil || res.Status != "dry_run" {
		t.Fatalf("indexnow Run() = %#v, %v", res, err)
	}
}

func symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
