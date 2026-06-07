package hooks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloudflarePurgeURLsDryRunSkipsHTTP(t *testing.T) {
	called := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		t.Fatal("cloudflare HTTP server must not be called in dry-run mode")
	}))
	defer srv.Close()

	tokenFile := writeSecretFile(t, "cloudflare_api_token", []byte("cf-secret-token"))
	client := NewCloudflareClient(Config{
		CloudflarePurgeEnabled: true,
		CloudflareZoneID:       "zone-id",
		CloudflareTokenFile:    tokenFile,
		HooksDryRun:            true,
		HooksMaxRetries:        3,
	}, srv.Client())

	res, err := client.PurgeURLs(context.Background(), []string{"https://example.com/posts/one/"})
	if err != nil {
		t.Fatalf("PurgeURLs() error = %v", err)
	}
	if res.Status != "dry_run" {
		t.Fatalf("expected dry_run status, got %q", res.Status)
	}
	if res.Provider != "cloudflare" {
		t.Fatalf("expected cloudflare provider, got %q", res.Provider)
	}
	if res.URLCount != 1 {
		t.Fatalf("expected URLCount 1, got %d", res.URLCount)
	}
	if called != 0 {
		t.Fatalf("expected no HTTP calls in dry-run mode, got %d", called)
	}
}

func TestCloudflarePurgeURLsRetriesAreBoundedAndRedacted(t *testing.T) {
	const secret = "cf-secret-token"
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.Header.Get("Authorization"); got != "Bearer "+secret {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "provider token leak: %s", secret)
	}))
	defer srv.Close()

	tokenFile := writeSecretFile(t, "cloudflare_api_token", []byte(secret))
	client := NewCloudflareClient(Config{
		CloudflarePurgeEnabled: true,
		CloudflareZoneID:       "zone-id",
		CloudflareTokenFile:    tokenFile,
		HooksDryRun:            false,
		HooksMaxRetries:        2,
	}, srv.Client())
	client.baseURL = srv.URL

	_, err := client.PurgeURLs(context.Background(), []string{"https://example.com/posts/one/"})
	if err == nil {
		t.Fatal("expected purge to fail")
	}
	if calls != 2 {
		t.Fatalf("expected 2 retry attempts, got %d", calls)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("expected error to redact secret, got %q", err.Error())
	}
}

func writeSecretFile(t *testing.T, name string, raw []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}
	return path
}
