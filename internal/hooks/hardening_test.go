package hooks

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromEnvRejectsNonPositiveRetries(t *testing.T) {
	t.Setenv("HUGO_HOOKS_MAX_RETRIES", "0")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("expected non-positive retry config to fail")
	}
}

func TestLoadSecretFileAcceptsStrictPermissionsAndRejectsTooOpen(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "token")
	if err := os.WriteFile(secret, []byte("super-secret\n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}
	if got, err := LoadSecretFile(secret, dir); err != nil || string(got) != "super-secret\n" {
		t.Fatalf("expected 0600 secret to load, got %q, %v", string(got), err)
	}
	if err := os.Chmod(secret, 0o640); err != nil {
		t.Fatalf("chmod secret 0640: %v", err)
	}
	if got, err := LoadSecretFile(secret, dir); err != nil || string(got) != "super-secret\n" {
		t.Fatalf("expected 0640 secret to load, got %q, %v", string(got), err)
	}
	if err := os.Chmod(secret, 0o666); err != nil {
		t.Fatalf("chmod secret world-readable: %v", err)
	}
	if _, err := LoadSecretFile(secret, dir); err == nil {
		t.Fatal("expected world-readable secret to fail")
	}
	if _, err := LoadSecretFile(secret, ""); err == nil {
		t.Fatal("expected missing allowed dir to fail")
	}
}

func TestStoreUtilityBranches(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if !store.HasColumn(ctx, "hook_jobs", "status") {
		t.Fatal("expected status column to exist")
	}
	if store.HasColumn(ctx, "nonexistent_table", "status") {
		t.Fatal("expected missing table column lookup to fail")
	}
	id1, err := store.Enqueue(ctx, HookJob{Provider: "cloudflare", Action: "URL_UPDATED", TargetURLs: []string{"https://example.com/a"}, Status: "pending"})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	id2, err := store.Enqueue(ctx, HookJob{Provider: "indexnow", Action: "URL_DELETED", TargetURLs: []string{"https://example.com/b"}, Status: "pending"})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if _, err := store.SetJobStatus(ctx, []string{id1, id2}, "failed"); err != nil {
		t.Fatalf("SetJobStatus() error = %v", err)
	}
	jobs, err := store.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if store.JobCount(ctx) != 2 {
		t.Fatalf("expected JobCount 2, got %d", store.JobCount(ctx))
	}
	if err := store.RecordAudit(ctx, AuditRecord{JobID: id1, Provider: "cloudflare", Action: "retry", Message: "retry queued"}); err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}
	audits := store.AuditMessages(ctx)
	if len(audits) != 1 || audits[0] != "retry queued" {
		t.Fatalf("unexpected audit messages: %#v", audits)
	}

	var nilStore *Store
	if nilStore.JobCount(ctx) != 0 {
		t.Fatal("expected nil store JobCount to return 0")
	}
	if msgs := nilStore.AuditMessages(ctx); msgs != nil {
		t.Fatalf("expected nil store AuditMessages to return nil, got %#v", msgs)
	}
}

func TestStoreNilAndDefaultBranches(t *testing.T) {
	ctx := context.Background()
	var nilStore *Store
	if _, err := nilStore.Enqueue(ctx, HookJob{Provider: "cloudflare", TargetURLs: []string{"https://example.com/a"}}); err == nil {
		t.Fatal("expected nil store enqueue to fail")
	}
	if _, err := nilStore.ListJobs(ctx); err == nil {
		t.Fatal("expected nil store list jobs to fail")
	}
	if _, err := nilStore.SetJobStatus(ctx, []string{"id"}, "pending"); err == nil {
		t.Fatal("expected nil store set job status to fail")
	}
	if err := nilStore.RecordAudit(ctx, AuditRecord{Action: "enqueue", Message: "x"}); err == nil {
		t.Fatal("expected nil store record audit to fail")
	}
	if nilStore.JobCount(ctx) != 0 {
		t.Fatal("expected nil store job count to return 0")
	}
	if msgs := nilStore.AuditMessages(ctx); msgs != nil {
		t.Fatalf("expected nil store audit messages to return nil, got %#v", msgs)
	}

	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	id, err := store.Enqueue(ctx, HookJob{Provider: "cloudflare", TargetURLs: []string{"https://example.com/a"}})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	jobs, err := store.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != "pending" {
		t.Fatalf("expected default pending status, got %q", jobs[0].Status)
	}
	if jobs[0].ID != id {
		t.Fatalf("expected job id %q, got %q", id, jobs[0].ID)
	}
}

func TestStoreErrorBranches(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "parent-file")
	if err := os.WriteFile(parent, []byte("occupied"), 0o600); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	if _, err := OpenStore(filepath.Join(parent, "hooks.db")); err == nil {
		t.Fatal("expected open store to fail when parent path is a file")
	}

	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if store.HasColumn(ctx, "hook_jobs)", "status") {
		t.Fatal("expected malformed table lookup to fail")
	}
	if _, err := store.Enqueue(ctx, HookJob{TargetURLs: []string{"https://example.com/a"}}); err == nil {
		t.Fatal("expected missing provider to fail")
	}
	if _, err := store.Enqueue(ctx, HookJob{Provider: "cloudflare"}); err == nil {
		t.Fatal("expected missing target urls to fail")
	}
	if err := store.RecordAudit(ctx, AuditRecord{Message: "missing action"}); err == nil {
		t.Fatal("expected missing action to fail")
	}
	if err := store.RecordAudit(ctx, AuditRecord{Action: "enqueue"}); err == nil {
		t.Fatal("expected missing message to fail")
	}

	origCreateSchema := createSchemaFunc
	createSchemaFunc = func(context.Context, *sql.DB) error {
		return fmt.Errorf("schema failed")
	}
	t.Cleanup(func() { createSchemaFunc = origCreateSchema })
	if _, err := OpenStore(filepath.Join(t.TempDir(), "schema-fail.db")); err == nil {
		t.Fatal("expected schema initialization failure")
	}
}

func TestCreateSchemaErrorBranch(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	origExec := execSchemaStatement
	execSchemaStatement = func(context.Context, *sql.DB, string) (sql.Result, error) {
		return nil, fmt.Errorf("schema exec failed")
	}
	t.Cleanup(func() { execSchemaStatement = origExec })

	if err := createSchema(context.Background(), store.db); err == nil {
		t.Fatal("expected schema exec failure")
	}
}

func TestCloudflareProviderBranches(t *testing.T) {
	var nilClient *CloudflareClient
	if _, err := nilClient.PurgeURLs(context.Background(), []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected nil cloudflare client to fail")
	}
	if _, err := NewCloudflareClient(Config{}, nil).PurgeURLs(context.Background(), []string{"ftp://example.com/a"}); err == nil {
		t.Fatal("expected invalid cloudflare URL to fail")
	}
	if err := validateHTTPSURLs([]string{""}); err == nil {
		t.Fatal("expected empty url to fail validation")
	}
	if err := validateHTTPSURLs([]string{"ftp://example.com/a"}); err == nil {
		t.Fatal("expected invalid url scheme to fail validation")
	}
	if maxRetries(0) != DefaultHooksMaxRetries {
		t.Fatalf("expected retry fallback %d, got %d", DefaultHooksMaxRetries, maxRetries(0))
	}
	if maxRetries(3) != 3 {
		t.Fatalf("expected retry passthrough 3, got %d", maxRetries(3))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewCloudflareClient(Config{CloudflarePurgeEnabled: true}, nil).PurgeURLs(ctx, []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected canceled cloudflare context to fail")
	}
	if _, err := NewCloudflareClient(Config{CloudflarePurgeEnabled: true, HooksDryRun: false, CloudflareZoneID: "zone", CloudflareTokenFile: filepath.Join(t.TempDir(), "missing")}, nil).PurgeURLs(context.Background(), []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected missing cloudflare token file to fail")
	}

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprint(w, `{"errors":[{"message":"rate limited"}]}`)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"success":true}`)
		}
	}))
	defer srv.Close()

	tokenFile := writeSecretFile(t, "cloudflare_api_token", []byte("cf-secret-token"))
	client := NewCloudflareClient(Config{
		CloudflarePurgeEnabled: true,
		CloudflareZoneID:       "zone-id",
		CloudflareTokenFile:    tokenFile,
		HooksDryRun:            false,
		HooksMaxRetries:        2,
	}, srv.Client())
	client.baseURL = srv.URL
	res, err := client.PurgeURLs(context.Background(), []string{"https://example.com/posts/one/"})
	if err != nil {
		t.Fatalf("PurgeURLs() error = %v", err)
	}
	if res.Status != "ok" || res.Attempts != 2 {
		t.Fatalf("unexpected cloudflare result: %#v", res)
	}
}

func TestGoogleIndexingProviderBranches(t *testing.T) {
	var nilClient *GoogleIndexingClient
	if _, err := nilClient.Publish(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected nil google client to fail")
	}
	serviceAccount := writeGoogleServiceAccount(t, "https://example.com/token", "service@example.com")
	client := NewGoogleIndexingClient(Config{
		GoogleIndexingEnabled:            true,
		GoogleIndexingServiceAccountFile: serviceAccount,
		HooksDryRun:                      false,
		HooksMaxRetries:                  2,
	}, nil)
	client.baseURL = "http://unused"
	if _, err := client.Publish(context.Background(), "URL_PUBLISHED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected invalid google indexing action to fail")
	}
	if _, err := client.Publish(context.Background(), "URL_UPDATED", nil); err == nil {
		t.Fatal("expected empty url list to fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Publish(ctx, "URL_UPDATED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected canceled google indexing context to fail")
	}
	badServiceAccount := filepath.Join(t.TempDir(), "bad-service-account.json")
	if err := os.WriteFile(badServiceAccount, []byte("{"), 0o600); err != nil {
		t.Fatalf("write bad service account: %v", err)
	}
	if _, err := NewGoogleIndexingClient(Config{
		GoogleIndexingEnabled:            true,
		GoogleIndexingServiceAccountFile: badServiceAccount,
		HooksDryRun:                      false,
		HooksMaxRetries:                  2,
	}, nil).Publish(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected invalid google service account to fail")
	}

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "google-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		default:
			calls++
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"urlNotificationMetadata":{"url":"https://example.com/a"}}`)
		}
	}))
	defer srv.Close()

	serviceAccount = writeGoogleServiceAccount(t, srv.URL+"/token", "service@example.com")
	client = NewGoogleIndexingClient(Config{
		GoogleIndexingEnabled:            true,
		GoogleIndexingServiceAccountFile: serviceAccount,
		HooksDryRun:                      false,
		HooksMaxRetries:                  2,
	}, srv.Client())
	client.baseURL = srv.URL
	client.tokenURL = srv.URL + "/token"
	res, err := client.Publish(context.Background(), "URL_UPDATED", []string{"https://example.com/a", "https://example.com/b"})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if res.Status != "ok" || res.Attempts != 2 || calls != 2 {
		t.Fatalf("unexpected google result: %#v calls=%d", res, calls)
	}
}

func TestIndexNowProviderBranches(t *testing.T) {
	var nilClient *IndexNowClient
	if _, err := nilClient.Submit(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected nil indexnow client to fail")
	}
	client := NewIndexNowClient(Config{IndexNowEnabled: true, HooksDryRun: false}, nil)
	if _, err := client.Submit(context.Background(), "URL_PUBLISHED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected invalid indexnow action to fail")
	}
	if _, err := client.Submit(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected missing endpoint configuration to fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Submit(ctx, "URL_UPDATED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected canceled indexnow context to fail")
	}
	if _, err := NewIndexNowClient(Config{IndexNowEnabled: true, HooksDryRun: false, IndexNowKeyFile: filepath.Join(t.TempDir(), "missing"), IndexNowEndpoint: "https://example.com/indexnow"}, nil).Submit(context.Background(), "URL_UPDATED", []string{"https://example.com/a"}); err == nil {
		t.Fatal("expected missing indexnow key file to fail")
	}

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	keyFile := writeIndexNowKey(t, "indexnow-secret-key")
	client = NewIndexNowClient(Config{
		IndexNowEnabled:  true,
		IndexNowKeyFile:  keyFile,
		IndexNowEndpoint: srv.URL,
		HooksDryRun:      false,
		HooksMaxRetries:  2,
	}, srv.Client())
	res, err := client.Submit(context.Background(), "URL_UPDATED", []string{"https://example.com/a", "https://example.com/b"})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if res.Status != "ok" || res.URLCount != 2 || calls != 1 {
		t.Fatalf("unexpected indexnow result: %#v calls=%d", res, calls)
	}
}

func TestPipelineNilAndDedupeBranches(t *testing.T) {
	var nilPipeline *Pipeline
	nilSummary := nilPipeline.Process(context.Background(), HookEvent{URLs: []string{"https://example.com/a"}})
	if nilSummary.HooksEnabled {
		t.Fatal("expected nil pipeline to report hooks disabled")
	}
	if nilSummary.QueuedURLsCount != 1 {
		t.Fatalf("expected queued url count 1, got %d", nilSummary.QueuedURLsCount)
	}

	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	providers := []HookProvider{
		&fakeHookProvider{name: "cloudflare", result: HookRunResult{Provider: "cloudflare", Status: "ok"}},
		&fakeHookProvider{name: "google_indexing", result: HookRunResult{Provider: "google_indexing", Status: "ok"}},
		&fakeHookProvider{name: "indexnow", result: HookRunResult{Provider: "indexnow", Status: "ok"}},
	}
	pipeline := NewPipeline(Config{PostBuildHooksEnabled: true}, store, providers...)
	summary := pipeline.Process(context.Background(), HookEvent{
		Mutation: "build_site",
		Action:   "URL_UPDATED",
		URLs: []string{
			"https://example.com/a",
			"https://example.com/a",
			" ",
			"https://example.com/b",
		},
	})
	if !summary.HooksEnabled {
		t.Fatal("expected hooks enabled")
	}
	if summary.QueuedURLsCount != 2 {
		t.Fatalf("expected deduped queued count 2, got %d", summary.QueuedURLsCount)
	}
	if store.JobCount(context.Background()) != 3 {
		t.Fatalf("expected one job per provider, got %d", store.JobCount(context.Background()))
	}
}

func TestPipelineEmptyURLsStillRunsProviders(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	provider := &fakeHookProvider{name: "cloudflare", result: HookRunResult{Provider: "cloudflare", Status: "ok"}}
	pipeline := NewPipeline(Config{PostBuildHooksEnabled: true}, store, provider)
	summary := pipeline.Process(context.Background(), HookEvent{
		Mutation: "build_site",
		Action:   "URL_UPDATED",
		URLs:     nil,
	})
	if summary.QueuedURLsCount != 0 {
		t.Fatalf("expected zero queued urls, got %d", summary.QueuedURLsCount)
	}
	if provider.calls != 1 {
		t.Fatalf("expected provider to run even with empty urls, got %d calls", provider.calls)
	}
	if store.JobCount(context.Background()) != 0 {
		t.Fatalf("expected no queued jobs for empty url set, got %d", store.JobCount(context.Background()))
	}
}

func TestRedactSecretsSkipsEmptyValues(t *testing.T) {
	got := redactSecrets("bearer token", "", "  ")
	if got != "bearer token" {
		t.Fatalf("expected no-op redact, got %q", got)
	}
}

func TestNewIDFallbackBranch(t *testing.T) {
	orig := randRead
	randRead = func([]byte) (int, error) {
		return 0, fmt.Errorf("entropy unavailable")
	}
	t.Cleanup(func() { randRead = orig })

	got := newID()
	if got != hex.EncodeToString([]byte("hooks-fallback-id")) {
		t.Fatalf("expected fallback id, got %q", got)
	}
}
