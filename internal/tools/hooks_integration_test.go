package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeHookPipeline struct {
	calls  []hooks.HookEvent
	summary hooks.HookSummary
}

func (f *fakeHookPipeline) Process(ctx context.Context, event hooks.HookEvent) hooks.HookSummary {
	f.calls = append(f.calls, event)
	if f.summary.QueuedURLsCount == 0 {
		f.summary.QueuedURLsCount = len(event.URLs)
	}
	return f.summary
}

func TestMutationToolsAttachHookSummaryAndEnqueueURLs(t *testing.T) {
	tests := []struct {
		name       string
		tool       string
		args       map[string]any
		wantURLs   []string
		wantAction string
	}{
		{
			name: "create_page",
			tool: "create_page",
			args: map[string]any{
				"route":   "/posts/hooks-test",
				"lang":    "fr",
				"title":   "Hooks test",
				"content": "Body",
			},
			wantURLs:   []string{"https://example.com/posts/hooks-test/"},
			wantAction: "URL_UPDATED",
		},
		{
			name: "update_page",
			tool: "update_page",
			args: map[string]any{
				"route":   "/posts/bonjour",
				"lang":    "fr",
				"title":   "Updated title",
				"content": "Updated body",
			},
			wantURLs:   []string{"https://example.com/posts/bonjour/"},
			wantAction: "URL_UPDATED",
		},
		{
			name: "delete_page",
			tool: "delete_page",
			args: map[string]any{
				"route": "/posts/plain",
			},
			wantURLs:   []string{"https://example.com/posts/plain/"},
			wantAction: "URL_DELETED",
		},
		{
			name: "upload_asset",
			tool: "upload_asset",
			args: map[string]any{
				"filename": "hooks-test.svg",
				"data":     base64.StdEncoding.EncodeToString([]byte("<svg/>")),
				"subfolder": "images/hooks",
			},
			wantURLs:   []string{"https://example.com/images/hooks/hooks-test.svg"},
			wantAction: "URL_UPDATED",
		},
		{
			name: "build_site",
			tool: "build_site",
			args: map[string]any{
				"purge_cf": false,
			},
			wantURLs:   []string{"https://example.com/"},
			wantAction: "URL_UPDATED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeHookPipeline{summary: hooks.HookSummary{
				HooksEnabled: true,
				CloudflarePurge: hooks.HookRunResult{
					Provider: "cloudflare",
					Status:   "ok",
				},
				GoogleIndexing: hooks.HookRunResult{
					Provider: "google_indexing",
					Status:   "ok",
				},
				IndexNow: hooks.HookRunResult{
					Provider: "indexnow",
					Status:   "ok",
				},
			}}
			deps := newMutationDeps(t, fakeBuildRunner{})
			deps.Hooks = fake
			deps.SiteBaseURL = "https://example.com"
			session, ctx := mustNewMutationSession(t, deps)
			defer session.Close()

			var toolArgs map[string]any = make(map[string]any)
			for k, v := range tc.args {
				toolArgs[k] = v
			}
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: toolArgs})
			if err != nil {
				t.Fatalf("CallTool(%s) error = %v", tc.tool, err)
			}
			if len(fake.calls) != 1 {
				t.Fatalf("expected 1 hook call, got %d", len(fake.calls))
			}
			if fake.calls[0].Action != tc.wantAction {
				t.Fatalf("unexpected hook action %q", fake.calls[0].Action)
			}
			if !equalStringSlices(fake.calls[0].URLs, tc.wantURLs) {
				t.Fatalf("unexpected hook URLs: got %#v want %#v", fake.calls[0].URLs, tc.wantURLs)
			}
			if len(res.Content) == 0 {
				t.Fatal("expected tool response content")
			}
			var response map[string]any
			text := res.Content[0].(*mcp.TextContent).Text
			if err := json.Unmarshal([]byte(text), &response); err != nil {
				t.Fatalf("response JSON error = %v", err)
			}
			if _, ok := response["hooks.enabled"]; !ok {
				t.Fatalf("response missing hooks summary: %s", text)
			}
			if _, ok := response["cloudflare_purge"]; !ok {
				t.Fatalf("response missing cloudflare_purge summary: %s", text)
			}
			if _, ok := response["google_indexing"]; !ok {
				t.Fatalf("response missing google_indexing summary: %s", text)
			}
			if _, ok := response["indexnow"]; !ok {
				t.Fatalf("response missing indexnow summary: %s", text)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
