package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/assets"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/pages"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
	"github.com/jmrGrav/hugo-mcp-go/internal/runner"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterExposesMutationToolsAndCallsWork(t *testing.T) {
	session, ctx := mustNewMutationSession(t, newMutationDeps(t, &fakeBuildRunner{}))
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	got := map[string]bool{}
	for _, tool := range toolsRes.Tools {
		got[tool.Name] = true
	}
	for _, name := range []string{"list_pages", "get_page", "list_assets", "create_page", "update_page", "delete_page", "upload_asset", "build_site"} {
		if !got[name] {
			t.Fatalf("tool %q not exposed", name)
		}
	}

	createRes := mustCallTool(t, session, ctx, "create_page", map[string]any{
		"route":   "/posts/sdk-phase2",
		"title":   "SDK Phase 2",
		"content": "Created through MCP.",
		"lang":    "fr",
	})
	assertToolResultStatus(t, createRes, "created")

	updateRes := mustCallTool(t, session, ctx, "update_page", map[string]any{
		"route":   "/posts/sdk-phase2",
		"lang":    "fr",
		"content": "Updated through MCP.",
	})
	assertToolResultStatus(t, updateRes, "updated")

	assetRes := mustCallTool(t, session, ctx, "upload_asset", map[string]any{
		"filename":  "sdk-phase2.svg",
		"data":      "PHN2ZyB2aWV3Qm94PSIwIDAgMSAxIi8+",
		"subfolder": "images/sdk",
	})
	assertToolResultStatus(t, assetRes, "ok")

	buildRes := mustCallTool(t, session, ctx, "build_site", map[string]any{"purge_cf": true})
	assertToolResultStatus(t, buildRes, "built")

	deleteRes := mustCallTool(t, session, ctx, "delete_page", map[string]any{
		"route": "/posts/sdk-phase2",
		"lang":  "fr",
	})
	assertToolResultStatus(t, deleteRes, "deleted")
}

func TestRegisterMutationToolErrorPaths(t *testing.T) {
	ws := newTestWorkspace(t)
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(ws.StaticRoot, "images", "escape")); err != nil {
		t.Fatal(err)
	}
	session, ctx := mustNewMutationSession(t, newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{}))
	defer session.Close()

	cases := []struct {
		name    string
		tool    string
		args    map[string]any
		wantSub string
	}{
		{name: "create missing title", tool: "create_page", args: map[string]any{"route": "/posts/x", "content": "body"}, wantSub: "required: missing properties"},
		{name: "create traversal", tool: "create_page", args: map[string]any{"route": "../escape", "title": "x", "content": "body"}, wantSub: "path traversal"},
		{name: "update missing route", tool: "update_page", args: map[string]any{"content": "body"}, wantSub: "required: missing properties"},
		{name: "delete missing route", tool: "delete_page", args: map[string]any{}, wantSub: "required: missing properties"},
		{name: "upload invalid base64", tool: "upload_asset", args: map[string]any{"filename": "a.svg", "data": "!!!"}, wantSub: "Invalid base64 data"},
		{name: "upload symlink", tool: "upload_asset", args: map[string]any{"filename": "a.svg", "data": "PHN2Zy8+", "subfolder": "images/escape"}, wantSub: "symlinks are not allowed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				t.Fatalf("CallTool(%s) protocol error = %v", tc.tool, err)
			}
			if tc.wantSub == "" {
				if res.IsError {
					t.Fatalf("CallTool(%s) unexpectedly returned error: %#v", tc.tool, res)
				}
				return
			}
			if !res.IsError {
				t.Fatalf("CallTool(%s) expected error response: %#v", tc.tool, res)
			}
			if len(res.Content) == 0 {
				t.Fatalf("CallTool(%s) error response had no content", tc.tool)
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.wantSub) {
				t.Fatalf("CallTool(%s) error text = %q want substring %q", tc.tool, text, tc.wantSub)
			}
		})
	}
}

func TestRegisterToolMetadataAndNilDependenciesAreComplete(t *testing.T) {
	session, ctx := mustNewMutationSession(t, Deps{})
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	gotNames := make([]string, 0, len(toolsRes.Tools))
	for _, tool := range toolsRes.Tools {
		gotNames = append(gotNames, tool.Name)
		if tool.Description == "" {
			t.Fatalf("tool %q missing description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Fatalf("tool %q missing input schema", tool.Name)
		}
	}
	wantOrder := []string{"build_site", "create_page", "delete_page", "get_page", "list_assets", "list_pages", "update_page", "upload_asset"}
	if !equalStrings(gotNames, wantOrder) {
		t.Fatalf("tool order = %#v want %#v", gotNames, wantOrder)
	}

	cases := []struct {
		tool    string
		args    map[string]any
		wantSub string
	}{
		{tool: "update_page", args: map[string]any{"route": "/posts/x", "content": "x"}, wantSub: "page mutation service not configured"},
		{tool: "delete_page", args: map[string]any{"route": "/posts/x"}, wantSub: "page mutation service not configured"},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				t.Fatalf("CallTool(%s) protocol error = %v", tc.tool, err)
			}
			if !res.IsError {
				t.Fatalf("CallTool(%s) expected error response: %#v", tc.tool, res)
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.wantSub) {
				t.Fatalf("CallTool(%s) error text = %q want substring %q", tc.tool, text, tc.wantSub)
			}
		})
	}
}

func TestRegisterReadOnlyToolServiceErrors(t *testing.T) {
	deps := newMutationDeps(t, &fakeBuildRunner{})
	deps.Pages = pages.New("")
	deps.Assets = assets.New("", "", "")
	session, ctx := mustNewMutationSession(t, deps)
	defer session.Close()

	cases := []struct {
		tool    string
		args    map[string]any
		wantSub string
	}{
		{tool: "list_pages", args: map[string]any{}, wantSub: "empty root"},
		{tool: "get_page", args: map[string]any{"route": "/posts/x"}, wantSub: "Page not found"},
		{tool: "list_assets", args: map[string]any{}, wantSub: "empty root"},
		{tool: "create_page", args: map[string]any{"route": "/posts/conflict", "title": "x", "content": "y", "frontmatter": map[string]any{"title": "x"}}, wantSub: "Conflict"},
		{tool: "update_page", args: map[string]any{"route": "/posts/bonjour", "lang": "fr", "frontmatter": map[string]any{"date": "2026-01-01T00:00:00Z"}}, wantSub: "cannot be modified"},
		{tool: "delete_page", args: map[string]any{"route": "/posts/missing", "lang": "fr"}, wantSub: "Page not found"},
	}

	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				t.Fatalf("CallTool(%s) protocol error = %v", tc.tool, err)
			}
			if !res.IsError {
				t.Fatalf("CallTool(%s) expected error response: %#v", tc.tool, res)
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.wantSub) {
				t.Fatalf("CallTool(%s) error text = %q want substring %q", tc.tool, text, tc.wantSub)
			}
		})
	}
}

func TestRegisterMutationToolProtocolAndSchemaEdges(t *testing.T) {
	session, ctx := mustNewMutationSession(t, newMutationDeps(t, &fakeBuildRunner{}))
	defer session.Close()

	cases := []struct {
		name    string
		tool    string
		args    map[string]any
		wantSub string
	}{
		{name: "unknown tool", tool: "does_not_exist", args: map[string]any{}, wantSub: "tool"},
		{name: "invalid type", tool: "create_page", args: map[string]any{"route": 123, "title": "x", "content": "body"}, wantSub: "type"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err == nil && (res == nil || !res.IsError) {
				t.Fatalf("CallTool(%s) unexpectedly succeeded: %#v", tc.tool, res)
			}
			if tc.wantSub == "" {
				return
			}
			if err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), tc.wantSub) {
					t.Fatalf("CallTool(%s) error = %v want substring %q", tc.tool, err, tc.wantSub)
				}
				return
			}
			if len(res.Content) == 0 {
				t.Fatalf("CallTool(%s) error response had no content", tc.tool)
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(strings.ToLower(text), tc.wantSub) {
				t.Fatalf("CallTool(%s) error text = %q want substring %q", tc.tool, text, tc.wantSub)
			}
		})
	}
}

func TestRegisterMutationToolUnknownFieldRejected(t *testing.T) {
	session, ctx := mustNewMutationSession(t, newMutationDeps(t, &fakeBuildRunner{}))
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_page",
		Arguments: map[string]any{
			"route":      "/posts/unknown-field",
			"title":      "Unknown Field",
			"content":    "Body",
			"unexpected": "ignored",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_page) protocol error = %v", err)
	}
	if !res.IsError {
		t.Fatalf("CallTool(create_page) unexpectedly accepted unknown field: %#v", res)
	}
}

func TestRegisterBuildToolErrorPaths(t *testing.T) {
	ws := newTestWorkspace(t)
	tests := []struct {
		name    string
		build   *mutations.BuildService
		wantSub string
	}{
		{
			name: "timeout",
			build: func() *mutations.BuildService {
				svc := mutations.NewBuildService(ws, &blockingRunner{})
				svc.Timeout = 20 * time.Millisecond
				return svc
			}(),
			wantSub: "context deadline exceeded",
		},
		{
			name: "runner failure",
			build: func() *mutations.BuildService {
				svc := mutations.NewBuildService(ws, &failingRunner{})
				svc.Timeout = time.Second
				return svc
			}(),
			wantSub: "build_site failed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session, ctx := mustNewMutationSession(t, Deps{
				Pages:          pages.New(ws.ContentRoot),
				Assets:         assets.New(ws.HugoRoot, ws.ContentRoot, ws.StaticRoot),
				PageMutations:  mutations.NewPageService(ws),
				AssetMutations: mutations.NewAssetService(ws),
				Build:          tc.build,
			})
			defer session.Close()
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "build_site", Arguments: map[string]any{"purge_cf": true}})
			if err != nil {
				t.Fatalf("CallTool(build_site) protocol error = %v", err)
			}
			if !res.IsError {
				t.Fatalf("CallTool(build_site) expected error response: %#v", res)
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.wantSub) {
				t.Fatalf("CallTool(build_site) error text = %q want substring %q", text, tc.wantSub)
			}
		})
	}
}

func TestMutationToolMissingStagingSerialization(t *testing.T) {
	session, ctx := mustNewMutationSession(t, Deps{
		Pages:          pages.New(""),
		Assets:         assets.New("", "", ""),
		PageMutations:  &mutations.PageService{},
		AssetMutations: &mutations.AssetService{},
		Build:          &mutations.BuildService{},
	})
	defer session.Close()

	cases := []struct {
		tool    string
		args    map[string]any
		wantSub string
	}{
		{tool: "create_page", args: map[string]any{"route": "/posts/x", "title": "x", "content": "x"}, wantSub: "missing staging workspace"},
		{tool: "upload_asset", args: map[string]any{"filename": "x.svg", "data": "PHN2Zy8+"}, wantSub: "missing staging workspace"},
		{tool: "build_site", args: map[string]any{}, wantSub: "missing staging workspace"},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				t.Fatalf("CallTool(%s) protocol error = %v", tc.tool, err)
			}
			if !res.IsError {
				t.Fatalf("CallTool(%s) expected error response: %#v", tc.tool, res)
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.wantSub) {
				t.Fatalf("CallTool(%s) error text = %q want substring %q", tc.tool, text, tc.wantSub)
			}
		})
	}
}

func TestMustJSON(t *testing.T) {
	if got := MustJSON(map[string]any{"a": 1}); got != `{"a":1}` {
		t.Fatalf("MustJSON() = %q", got)
	}
}

func TestRegisterNilDependencyErrors(t *testing.T) {
	session, ctx := mustNewMutationSession(t, Deps{})
	defer session.Close()

	cases := []struct {
		tool    string
		args    map[string]any
		wantSub string
	}{
		{tool: "list_pages", args: map[string]any{}, wantSub: "pages service not configured"},
		{tool: "get_page", args: map[string]any{"route": "/posts/x"}, wantSub: "pages service not configured"},
		{tool: "list_assets", args: map[string]any{}, wantSub: "assets service not configured"},
		{tool: "create_page", args: map[string]any{"route": "/posts/x", "title": "x", "content": "x"}, wantSub: "page mutation service not configured"},
		{tool: "upload_asset", args: map[string]any{"filename": "x.svg", "data": "PHN2Zy8+"}, wantSub: "asset mutation service not configured"},
		{tool: "build_site", args: map[string]any{}, wantSub: "build service not configured"},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				t.Fatalf("CallTool(%s) protocol error = %v", tc.tool, err)
			}
			if !res.IsError {
				t.Fatalf("CallTool(%s) expected error response: %#v", tc.tool, res)
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.wantSub) {
				t.Fatalf("CallTool(%s) error text = %q want substring %q", tc.tool, text, tc.wantSub)
			}
		})
	}
}

func mustCallTool(t *testing.T, session *mcp.ClientSession, ctx context.Context, name string, args map[string]any) map[string]any {
	t.Helper()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("CallTool(%s) returned no content", name)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("CallTool(%s) JSON decode error = %v", name, err)
	}
	return got
}

func assertToolResultStatus(t *testing.T, got map[string]any, want string) {
	t.Helper()
	if got["status"] != want {
		t.Fatalf("status = %v want %q", got["status"], want)
	}
}

func equalStrings(a, b []string) bool {
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

func mustNewMutationSession(t *testing.T, deps Deps) (*mcp.ClientSession, context.Context) {
	t.Helper()
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
	return session, ctx
}

func newMutationDeps(t *testing.T, buildRunner runner.Runner) Deps {
	t.Helper()
	ws := newTestWorkspace(t)
	return newMutationDepsWithWorkspace(t, ws, buildRunner)
}

func newMutationDepsWithWorkspace(t *testing.T, ws *staging.Workspace, buildRunner runner.Runner) Deps {
	t.Helper()
	return Deps{
		Pages:          pages.New(ws.ContentRoot),
		Assets:         assets.New(ws.HugoRoot, ws.ContentRoot, ws.StaticRoot),
		PageMutations:  mutations.NewPageService(ws),
		AssetMutations: mutations.NewAssetService(ws),
		Build:          mutations.NewBuildService(ws, buildRunner),
	}
}

func newTestWorkspace(t *testing.T) *staging.Workspace {
	t.Helper()
	root := t.TempDir()
	content := filepath.Join(root, "content")
	static := filepath.Join(root, "static")
	public := filepath.Join(root, "public")
	work := filepath.Join(root, "work")
	copyDir(t, filepath.Join("..", "..", "testdata", "fixtures", "minimal-site", "content"), content)
	copyDir(t, filepath.Join("..", "..", "testdata", "fixtures", "minimal-site", "static"), static)
	for _, dir := range []string{public, work} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := staging.New(root, content, static, public, work)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		raw, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dstPath, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

type fakeBuildRunner struct{}

func (fakeBuildRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	return "", "", nil
}

type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	<-ctx.Done()
	return "", "", ctx.Err()
}

type failingRunner struct{}

func (failingRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	return "", "boom", context.Canceled
}

var _ runner.Runner = fakeBuildRunner{}
