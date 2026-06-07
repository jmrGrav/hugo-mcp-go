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

func TestIndexNowDryRunSkipsHTTP(t *testing.T) {
	called := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		t.Fatal("indexnow HTTP server must not be called in dry-run mode")
	}))
	defer srv.Close()

	keyFile := writeIndexNowKey(t, "indexnow-secret-key")
	client := NewIndexNowClient(Config{
		IndexNowEnabled: true,
		IndexNowKeyFile: keyFile,
		IndexNowEndpoint: srv.URL,
		HooksDryRun: true,
		HooksMaxRetries: 3,
	}, srv.Client())

	res, err := client.Submit(context.Background(), "URL_UPDATED", []string{"https://example.com/posts/one/"})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if res.Status != "dry_run" {
		t.Fatalf("expected dry_run status, got %q", res.Status)
	}
	if res.Provider != "indexnow" {
		t.Fatalf("expected indexnow provider, got %q", res.Provider)
	}
	if called != 0 {
		t.Fatalf("expected no HTTP calls in dry-run mode, got %d", called)
	}
}

func TestIndexNowRetriesAreBoundedAndRedacted(t *testing.T) {
	const secret = "indexnow-secret-key"
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "indexnow key leak: %s", secret)
	}))
	defer srv.Close()

	keyFile := writeIndexNowKey(t, secret)
	client := NewIndexNowClient(Config{
		IndexNowEnabled: true,
		IndexNowKeyFile: keyFile,
		IndexNowEndpoint: srv.URL,
		HooksDryRun: false,
		HooksMaxRetries: 2,
	}, srv.Client())

	_, err := client.Submit(context.Background(), "URL_UPDATED", []string{"https://example.com/posts/one/"})
	if err == nil {
		t.Fatal("expected submit to fail")
	}
	if calls != 2 {
		t.Fatalf("expected 2 retry attempts, got %d", calls)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("expected error to redact secret, got %q", err.Error())
	}
}

func writeIndexNowKey(t *testing.T, secret string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "indexnow.key")
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		t.Fatalf("write indexnow key: %v", err)
	}
	return path
}
