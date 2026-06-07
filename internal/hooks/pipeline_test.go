package hooks

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

type fakeHookProvider struct {
	name   string
	result HookRunResult
	err    error
	calls  int
	action string
	urls   []string
}

func (f *fakeHookProvider) Name() string { return f.name }

func (f *fakeHookProvider) Run(ctx context.Context, action string, urls []string) (HookRunResult, error) {
	f.calls++
	f.action = action
	f.urls = append([]string(nil), urls...)
	return f.result, f.err
}

func TestPipelineQueuesPendingJobsWhenHooksDisabled(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	provider := &fakeHookProvider{name: "cloudflare", result: HookRunResult{Provider: "cloudflare", Status: "ok"}}
	pipeline := NewPipeline(Config{PostBuildHooksEnabled: false}, store, provider)

	summary := pipeline.Process(context.Background(), HookEvent{
		Mutation: "update_page",
		Action:   "URL_UPDATED",
		URLs:     []string{"https://example.com/posts/one/"},
	})

	if summary.HooksEnabled {
		t.Fatal("expected hooks to be disabled")
	}
	if summary.QueuedURLsCount != 1 {
		t.Fatalf("expected 1 queued URL, got %d", summary.QueuedURLsCount)
	}
	if summary.FailedJobsCount != 0 {
		t.Fatalf("expected 0 failed jobs, got %d", summary.FailedJobsCount)
	}
	if provider.calls != 0 {
		t.Fatalf("expected provider not to run while hooks are disabled, got %d calls", provider.calls)
	}
	if got := store.JobCount(context.Background()); got != 1 {
		t.Fatalf("expected 1 queued job, got %d", got)
	}
}

func TestPipelineRunsProvidersWhenHooksEnabled(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	provider := &fakeHookProvider{name: "cloudflare", result: HookRunResult{Provider: "cloudflare", Status: "ok", URLCount: 1, Attempts: 1}}
	pipeline := NewPipeline(Config{PostBuildHooksEnabled: true}, store, provider)

	summary := pipeline.Process(context.Background(), HookEvent{
		Mutation: "build_site",
		Action:   "URL_UPDATED",
		URLs:     []string{"https://example.com/posts/one/"},
	})

	if !summary.HooksEnabled {
		t.Fatal("expected hooks to be enabled")
	}
	if summary.CloudflarePurge.Status != "ok" {
		t.Fatalf("expected cloudflare status ok, got %q", summary.CloudflarePurge.Status)
	}
	if provider.calls != 1 {
		t.Fatalf("expected provider to run once, got %d calls", provider.calls)
	}
	if provider.action != "URL_UPDATED" {
		t.Fatalf("unexpected action %q", provider.action)
	}
	if len(provider.urls) != 1 || provider.urls[0] != "https://example.com/posts/one/" {
		t.Fatalf("unexpected provider URLs: %#v", provider.urls)
	}
	if got := store.JobCount(context.Background()); got != 1 {
		t.Fatalf("expected 1 queued job, got %d", got)
	}
}

func TestPipelineRecordsRedactedFailureWithoutBreakingSummary(t *testing.T) {
	const secret = "top-secret-token"
	store, err := OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	provider := &fakeHookProvider{name: "cloudflare", err: fmt.Errorf("provider failed: Bearer %s", secret)}
	pipeline := NewPipeline(Config{PostBuildHooksEnabled: true}, store, provider)

	summary := pipeline.Process(context.Background(), HookEvent{
		Mutation: "delete_page",
		Action:   "URL_DELETED",
		URLs:     []string{"https://example.com/posts/one/"},
	})

	if summary.FailedJobsCount != 1 {
		t.Fatalf("expected 1 failed job, got %d", summary.FailedJobsCount)
	}
	if provider.calls != 1 {
		t.Fatalf("expected provider to run once, got %d calls", provider.calls)
	}
	audits := store.AuditMessages(context.Background())
	if len(audits) == 0 {
		t.Fatal("expected audit records")
	}
	if strings.Contains(strings.Join(audits, "\n"), secret) {
		t.Fatalf("expected audit messages to redact secret, got %#v", audits)
	}
}
