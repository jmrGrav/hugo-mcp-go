package hooks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoogleIndexingDryRunSkipsHTTP(t *testing.T) {
	called := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		t.Fatal("google indexing HTTP server must not be called in dry-run mode")
	}))
	defer srv.Close()

	serviceAccount := writeGoogleServiceAccount(t, srv.URL+"/token", "service@example.com")
	client := NewGoogleIndexingClient(Config{
		GoogleIndexingEnabled:            true,
		GoogleIndexingServiceAccountFile: serviceAccount,
		HooksDryRun:                      true,
		HooksMaxRetries:                  3,
	}, srv.Client())
	client.baseURL = srv.URL
	client.tokenURL = srv.URL + "/token"

	res, err := client.Publish(context.Background(), "URL_UPDATED", []string{"https://example.com/posts/one/"})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if res.Status != "dry_run" {
		t.Fatalf("expected dry_run status, got %q", res.Status)
	}
	if res.Provider != "google_indexing" {
		t.Fatalf("expected google_indexing provider, got %q", res.Provider)
	}
	if res.URLCount != 1 {
		t.Fatalf("expected URLCount 1, got %d", res.URLCount)
	}
	if called != 0 {
		t.Fatalf("expected no HTTP calls in dry-run mode, got %d", called)
	}
}

func TestGoogleIndexingRetriesAreBoundedAndRedacted(t *testing.T) {
	const accessToken = "google-access-token"
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": accessToken,
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		default:
			calls++
			if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
				t.Fatalf("unexpected Authorization header: %q", got)
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "google token leak: %s", accessToken)
		}
	}))
	defer srv.Close()

	serviceAccount := writeGoogleServiceAccount(t, srv.URL+"/token", "service@example.com")
	client := NewGoogleIndexingClient(Config{
		GoogleIndexingEnabled:            true,
		GoogleIndexingServiceAccountFile: serviceAccount,
		HooksDryRun:                      false,
		HooksMaxRetries:                  2,
	}, srv.Client())
	client.baseURL = srv.URL
	client.tokenURL = srv.URL + "/token"

	_, err := client.Publish(context.Background(), "URL_UPDATED", []string{"https://example.com/posts/one/"})
	if err == nil {
		t.Fatal("expected publish to fail")
	}
	if calls != 2 {
		t.Fatalf("expected 2 retry attempts, got %d", calls)
	}
	if strings.Contains(err.Error(), accessToken) {
		t.Fatalf("expected error to redact access token, got %q", err.Error())
	}
}

func writeGoogleServiceAccount(t *testing.T, tokenURI, email string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	doc := map[string]any{
		"type":         "service_account",
		"project_id":   "test-project",
		"private_key_id": "test-key-id",
		"private_key":  string(pemBytes),
		"client_email": email,
		"token_uri":    tokenURI,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal service account: %v", err)
	}
	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write service account: %v", err)
	}
	return path
}
