package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHookAdminToolsAreOptIn(t *testing.T) {
	deps := newMutationDeps(t, fakeBuildRunner{})
	deps.Hooks = &fakeHookPipeline{summary: hooks.HookSummary{HooksEnabled: true}}
	deps.HooksStore = nil
	deps.HooksAdminEnabled = false

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.1"}, nil)
	Register(server, deps)
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server connect error = %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect error = %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	got := map[string]bool{}
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}
	for _, name := range []string{"list_hook_jobs", "retry_hook_jobs", "get_hook_status", "run_post_build_hooks"} {
		if got[name] {
			t.Fatalf("admin tool %q should be hidden when disabled", name)
		}
	}
}

func TestHookAdminToolsListRetryStatusAndRunAreSanitized(t *testing.T) {
	const secret = "Bearer admin-secret"
	store, err := hooks.OpenStore(filepath.Join(t.TempDir(), "hooks.db"))
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	jobID, err := store.Enqueue(context.Background(), hooks.HookJob{
		Provider:   "cloudflare",
		Action:     "URL_UPDATED",
		TargetURLs: []string{"https://example.com/posts/one/"},
		Status:     "failed",
		LastError:  secret,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if jobID == "" {
		t.Fatal("expected job id")
	}
	fake := &fakeHookPipeline{summary: hooks.HookSummary{
		HooksEnabled: true,
		CloudflarePurge: hooks.HookRunResult{
			Provider: "cloudflare",
			Status:   "ok",
		},
	}}
	deps := newMutationDeps(t, fakeBuildRunner{})
	deps.Hooks = fake
	deps.HooksStore = store
	deps.HooksAdminEnabled = true
	deps.SiteBaseURL = "https://example.com"

	session, ctx := mustNewMutationSession(t, deps)
	defer session.Close()

	listRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_hook_jobs", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("list_hook_jobs error = %v", err)
	}
	listPayload := decodeToolJSON(t, listRes)
	if got := listPayload["total"]; got != float64(1) {
		t.Fatalf("expected 1 listed job, got %#v", got)
	}
	jobs := listPayload["jobs"].([]any)
	first := jobs[0].(map[string]any)
	if strings.Contains(first["last_error"].(string), "admin-secret") {
		t.Fatalf("expected last_error to be redacted, got %#v", first["last_error"])
	}

	statusRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_hook_status", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("get_hook_status error = %v", err)
	}
	statusPayload := decodeToolJSON(t, statusRes)
	if statusPayload["hooks.enabled"] != true {
		t.Fatalf("expected hooks.enabled true, got %#v", statusPayload["hooks.enabled"])
	}

	runRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "run_post_build_hooks", Arguments: map[string]any{
		"action": "URL_UPDATED",
		"urls":   []string{"https://example.com/posts/one/"},
	}})
	if err != nil {
		t.Fatalf("run_post_build_hooks error = %v", err)
	}
	runPayload := decodeToolJSON(t, runRes)
	if runPayload["hooks.enabled"] != true {
		t.Fatalf("expected run summary hooks.enabled true, got %#v", runPayload["hooks.enabled"])
	}
	if len(fake.calls) == 0 {
		t.Fatal("expected fake pipeline to be invoked")
	}

	retryRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "retry_hook_jobs", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("retry_hook_jobs error = %v", err)
	}
	retryPayload := decodeToolJSON(t, retryRes)
	if retryPayload["retried_count"] != float64(1) {
		t.Fatalf("expected retried_count 1, got %#v", retryPayload["retried_count"])
	}
}

func decodeToolJSON(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("tool response missing content")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("decode tool response: %v", err)
	}
	return out
}
