package hooks

import "testing"

func TestHookSummaryMCPUsesSanitizedShape(t *testing.T) {
	summary := HookSummary{
		HooksEnabled:    true,
		QueuedURLsCount: 2,
		FailedJobsCount: 1,
		CloudflarePurge: HookRunResult{Provider: "cloudflare", Status: "ok", URLCount: 2, Attempts: 1, DryRun: false, Message: "secret should not escape"},
	}

	got := summary.MCP()
	if got["hooks.enabled"] != true {
		t.Fatalf("expected hooks.enabled true, got %#v", got["hooks.enabled"])
	}
	if got["queued_urls_count"] != 2 {
		t.Fatalf("expected queued_urls_count 2, got %#v", got["queued_urls_count"])
	}
	if got["failed_jobs_count"] != 1 {
		t.Fatalf("expected failed_jobs_count 1, got %#v", got["failed_jobs_count"])
	}
	purge := got["cloudflare_purge"].(map[string]any)
	if _, ok := purge["message"]; ok {
		t.Fatal("message must not be exposed in MCP summary")
	}
	if purge["status"] != "ok" {
		t.Fatalf("expected cloudflare status ok, got %#v", purge["status"])
	}
}
