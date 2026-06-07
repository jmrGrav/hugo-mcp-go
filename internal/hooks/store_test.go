package hooks

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenStoreCreatesStateTablesWithoutSecrets(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	for _, table := range []string{"hook_jobs", "hook_attempts", "hook_provider_state", "hook_audit"} {
		if !store.HasTable(ctx, table) {
			t.Fatalf("expected table %q to exist", table)
		}
	}

	if store.HasColumn(ctx, "hook_jobs", "cloudflare_token") {
		t.Fatal("secret column cloudflare_token must not exist")
	}
	if store.HasColumn(ctx, "hook_jobs", "google_private_key") {
		t.Fatal("secret column google_private_key must not exist")
	}
	if store.HasColumn(ctx, "hook_jobs", "indexnow_key") {
		t.Fatal("secret column indexnow_key must not exist")
	}
}

func TestStoreRejectsSecretMaterial(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	jobID, err := store.Enqueue(ctx, HookJob{
		Provider:   "cloudflare",
		TargetURLs: []string{"https://example.com/post/"},
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if jobID == "" {
		t.Fatal("expected a job id")
	}
	if err := store.RecordAudit(ctx, AuditRecord{
		JobID:   jobID,
		Action:  "enqueue",
		Message: "queued without bearer token leakage",
	}); err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}
}
