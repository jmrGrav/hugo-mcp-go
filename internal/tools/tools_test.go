package tools

import (
	"context"
	"encoding/json"
	"fmt"
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
	for _, name := range []string{"list_pages", "get_page", "get_page_chunk", "list_assets", "get_asset_chunk", "create_page", "update_page", "delete_page", "upload_asset", "build_site"} {
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

func TestRegisterPaginationAndChunkTools(t *testing.T) {
	ws := newTestWorkspace(t)
	if err := os.MkdirAll(filepath.Join(ws.ContentRoot, "posts", "chunked"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.ContentRoot, "posts", "chunked", "index.fr.md"), []byte("---\ntitle: chunked\n---\n"+strings.Repeat("abcdef", 32)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws.StaticRoot, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.StaticRoot, "images", "chunk.bin"), []byte("0123456789abcdef0123456789abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}

	session, ctx := mustNewMutationSession(t, Deps{
		Pages:          pages.New(ws.ContentRoot),
		Assets:         assets.New(ws.HugoRoot, ws.ContentRoot, ws.StaticRoot),
		PageMutations:  mutations.NewPageService(ws),
		AssetMutations: mutations.NewAssetService(ws),
		Build:          mutations.NewBuildService(ws, &fakeBuildRunner{}),
	})
	defer session.Close()

	listPages := mustCallToolResult(t, session, ctx, "list_pages", map[string]any{"limit": 1})
	pagesOut := mustStructuredMap(t, listPages)
	if v, ok := pagesOut["has_more"].(bool); !ok || !v {
		t.Fatalf("list_pages expected has_more=true: %#v", pagesOut)
	}
	if pagesOut["next_cursor"] == "" {
		t.Fatalf("list_pages expected next_cursor: %#v", pagesOut)
	}

	pageChunk := mustCallToolResult(t, session, ctx, "get_page_chunk", map[string]any{"route": "/posts/chunked", "lang": "fr", "cursor": 0, "chunk_bytes": 16})
	pageChunkOut := mustStructuredMap(t, pageChunk)
	if pageChunkOut["chunk"] == "" {
		t.Fatalf("get_page_chunk returned empty chunk: %#v", pageChunkOut)
	}
	if pageChunkOut["next_cursor"] == "" {
		t.Fatalf("get_page_chunk expected next_cursor: %#v", pageChunkOut)
	}

	assetChunk := mustCallToolResult(t, session, ctx, "get_asset_chunk", map[string]any{"path": "static/images/chunk.bin", "cursor": 0, "chunk_bytes": 8})
	assetChunkOut := mustStructuredMap(t, assetChunk)
	if assetChunkOut["chunk"] == "" {
		t.Fatalf("get_asset_chunk returned empty chunk: %#v", assetChunkOut)
	}
	if assetChunkOut["next_cursor"] == "" {
		t.Fatalf("get_asset_chunk expected next_cursor: %#v", assetChunkOut)
	}
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
	toolsByName := make(map[string]*mcp.Tool, len(toolsRes.Tools))
	for _, tool := range toolsRes.Tools {
		gotNames = append(gotNames, tool.Name)
		toolsByName[tool.Name] = tool
		if tool.Description == "" {
			t.Fatalf("tool %q missing description", tool.Name)
		}
		if tool.Title == "" {
			t.Fatalf("tool %q missing title", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Fatalf("tool %q missing input schema", tool.Name)
		}
	}
	wantOrder := []string{"build_site", "check_sri_versions", "create_page", "delete_page", "generate_featured_image", "get_asset_chunk", "get_page", "get_page_chunk", "list_assets", "list_pages", "update_page", "upload_asset"}
	if !equalStrings(gotNames, wantOrder) {
		t.Fatalf("tool order = %#v want %#v", gotNames, wantOrder)
	}

	readOnlyTools := []string{"get_asset_chunk", "get_page", "get_page_chunk", "list_assets", "list_pages"}
	for _, name := range readOnlyTools {
		tool := toolsByName[name]
		if tool == nil {
			t.Fatalf("tool %q missing", name)
		}
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatalf("tool %q expected readOnlyHint=true, got %#v", name, tool.Annotations)
		}
	}

	destructiveTools := []string{"build_site", "check_sri_versions", "create_page", "delete_page", "generate_featured_image", "update_page", "upload_asset"}
	for _, name := range destructiveTools {
		tool := toolsByName[name]
		if tool == nil {
			t.Fatalf("tool %q missing", name)
		}
		if tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
			t.Fatalf("tool %q expected destructiveHint=true, got %#v", name, tool.Annotations)
		}
	}

	listAssetsSchema := schemaProperty(t, toolsByName["list_assets"], "path_prefix")
	if desc := stringValue(listAssetsSchema["description"]); !strings.Contains(desc, "exact directory prefix") {
		t.Fatalf("list_assets.path_prefix description = %q", desc)
	}
	imageSchema := schemaProperty(t, toolsByName["generate_featured_image"], "accent")
	if desc := stringValue(imageSchema["description"]); !strings.Contains(desc, "#7aa2f7") {
		t.Fatalf("generate_featured_image.accent description = %q", desc)
	}

	hookSession, hookCtx := mustNewMutationSession(t, Deps{HooksAdminEnabled: true})
	defer hookSession.Close()
	hookToolsRes, err := hookSession.ListTools(hookCtx, nil)
	if err != nil {
		t.Fatalf("ListTools() hooks error = %v", err)
	}
	hookTools := map[string]*mcp.Tool{}
	for _, tool := range hookToolsRes.Tools {
		hookTools[tool.Name] = tool
	}
	for _, name := range []string{"list_hook_jobs", "get_hook_status"} {
		tool := hookTools[name]
		if tool == nil || tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatalf("hook tool %q expected readOnlyHint=true, got %#v", name, tool)
		}
	}
	for _, name := range []string{"retry_hook_jobs", "run_post_build_hooks"} {
		tool := hookTools[name]
		if tool == nil || tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
			t.Fatalf("hook tool %q expected destructiveHint=true, got %#v", name, tool)
		}
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

func schemaProperty(t *testing.T, tool *mcp.Tool, field string) map[string]any {
	t.Helper()
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema for %s: %v", tool.Name, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode input schema for %s: %v", tool.Name, err)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("tool %s missing properties: %#v", tool.Name, schema)
	}
	prop, ok := props[field].(map[string]any)
	if !ok {
		t.Fatalf("tool %s missing property %q: %#v", tool.Name, field, schema)
	}
	return prop
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
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

// TestSchemaPropertiesHaveExplicitTypes is a targeted regression test for the
// three fields that previously used Go's `any` type and produced empty {}
// JSON Schema fragments (no type/oneOf/anyOf/$ref), which Claude Code rejects.
func TestSchemaPropertiesHaveExplicitTypes(t *testing.T) {
	session, ctx := mustNewMutationSession(t, Deps{})
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	toolsByName := make(map[string]*mcp.Tool, len(toolsRes.Tools))
	for _, tool := range toolsRes.Tools {
		toolsByName[tool.Name] = tool
	}

	cases := []struct {
		tool     string
		schema   string // "input" or "output"
		property string
	}{
		{tool: "create_page", schema: "input", property: "frontmatter"},
		{tool: "update_page", schema: "input", property: "frontmatter"},
		{tool: "check_sri_versions", schema: "output", property: "downstream"},
	}

	for _, tc := range cases {
		t.Run(tc.tool+"/"+tc.schema+"/"+tc.property, func(t *testing.T) {
			tool := toolsByName[tc.tool]
			if tool == nil {
				t.Fatalf("tool %q not found", tc.tool)
			}

			var src any
			if tc.schema == "input" {
				src = tool.InputSchema
			} else {
				src = tool.OutputSchema
			}
			if src == nil {
				t.Fatalf("tool %q %s schema is nil", tc.tool, tc.schema)
			}

			raw, err := json.Marshal(src)
			if err != nil {
				t.Fatalf("marshal %s schema for %s: %v", tc.schema, tc.tool, err)
			}
			var schema map[string]any
			if err := json.Unmarshal(raw, &schema); err != nil {
				t.Fatalf("decode %s schema for %s: %v", tc.schema, tc.tool, err)
			}
			props, ok := schema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("tool %s %s schema missing properties", tc.tool, tc.schema)
			}
			prop, ok := props[tc.property].(map[string]any)
			if !ok {
				t.Fatalf("tool %s %s schema missing property %q", tc.tool, tc.schema, tc.property)
			}
			typ, ok := prop["type"].(string)
			if !ok || typ == "" {
				t.Fatalf("tool %s %s schema property %q has no explicit type (got %#v); Claude Code rejects empty schemas", tc.tool, tc.schema, tc.property, prop)
			}
		})
	}
}

// TestAllToolSchemasHaveNoEmptyFragments walks every inputSchema and
// outputSchema for every registered tool and fails if any JSON Schema node
// is an empty object ({}) without a type, oneOf, anyOf, allOf, $ref, or
// enum keyword. Such nodes are generated when Go's `any`/`interface{}` type
// leaks into a schema and are rejected by the Claude Code MCP validator.
func TestAllToolSchemasHaveNoEmptyFragments(t *testing.T) {
	session, ctx := mustNewMutationSession(t, Deps{})
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	for _, tool := range toolsRes.Tools {
		if tool.InputSchema != nil {
			raw, err := json.Marshal(tool.InputSchema)
			if err != nil {
				t.Fatalf("tool %s: marshal inputSchema: %v", tool.Name, err)
			}
			var node map[string]any
			if err := json.Unmarshal(raw, &node); err != nil {
				t.Fatalf("tool %s: decode inputSchema: %v", tool.Name, err)
			}
			if violations := collectEmptySchemaNodes(node, "inputSchema"); len(violations) > 0 {
				for _, v := range violations {
					t.Errorf("tool %s: empty schema node at %s — add a type or constraint (Claude Code rejects {})", tool.Name, v)
				}
			}
		}
		if tool.OutputSchema != nil {
			raw, err := json.Marshal(tool.OutputSchema)
			if err != nil {
				t.Fatalf("tool %s: marshal outputSchema: %v", tool.Name, err)
			}
			var node map[string]any
			if err := json.Unmarshal(raw, &node); err != nil {
				t.Fatalf("tool %s: decode outputSchema: %v", tool.Name, err)
			}
			if violations := collectEmptySchemaNodes(node, "outputSchema"); len(violations) > 0 {
				for _, v := range violations {
					t.Errorf("tool %s: empty schema node at %s — add a type or constraint (Claude Code rejects {})", tool.Name, v)
				}
			}
		}
	}
}

// schemaKeywords is the set of JSON Schema keywords that give a node meaning.
// A node that has none of these is considered an empty/unconstrained fragment.
var schemaKeywords = map[string]struct{}{
	"type": {}, "oneOf": {}, "anyOf": {}, "allOf": {}, "$ref": {},
	"enum": {}, "const": {}, "not": {}, "if": {},
}

// collectEmptySchemaNodes recursively walks a decoded JSON Schema object and
// returns the JSON pointer paths of any nodes that are empty objects (no
// recognised keywords). It skips meta-fields like "title", "description",
// "default", "$schema", "definitions", "$defs", and "examples".
func collectEmptySchemaNodes(node map[string]any, path string) []string {
	skipKeys := map[string]struct{}{
		"title": {}, "description": {}, "default": {}, "$schema": {},
		"definitions": {}, "$defs": {}, "examples": {}, "deprecated": {},
	}

	var violations []string

	hasKeyword := false
	for k := range schemaKeywords {
		if _, ok := node[k]; ok {
			hasKeyword = true
			break
		}
	}
	meaningfulKeys := 0
	for k := range node {
		if _, skip := skipKeys[k]; !skip {
			meaningfulKeys++
		}
	}
	if !hasKeyword && meaningfulKeys == 0 {
		violations = append(violations, path)
	}

	// Recurse into properties
	if props, ok := node["properties"].(map[string]any); ok {
		for name, val := range props {
			if child, ok := val.(map[string]any); ok {
				violations = append(violations, collectEmptySchemaNodes(child, path+"/properties/"+name)...)
			}
		}
	}

	// Recurse into additionalProperties if it's an object (not a bool)
	if ap, ok := node["additionalProperties"].(map[string]any); ok {
		violations = append(violations, collectEmptySchemaNodes(ap, path+"/additionalProperties")...)
	}

	// Recurse into items
	if items, ok := node["items"].(map[string]any); ok {
		violations = append(violations, collectEmptySchemaNodes(items, path+"/items")...)
	}

	// Recurse into oneOf / anyOf / allOf arrays
	for _, keyword := range []string{"oneOf", "anyOf", "allOf"} {
		if arr, ok := node[keyword].([]any); ok {
			for i, elem := range arr {
				if child, ok := elem.(map[string]any); ok {
					violations = append(violations, collectEmptySchemaNodes(child, fmt.Sprintf("%s/%s/%d", path, keyword, i))...)
				}
			}
		}
	}

	return violations
}

// TestToolsListClientView validates the complete tools/list payload as seen by
// a real MCP client session. It goes beyond TestAllToolSchemasHaveNoEmptyFragments
// (which inspects Go structures) by testing the full wire path:
//
//  1. ListTools is called over an in-process MCP transport.
//  2. The entire result is marshalled to JSON and unmarshalled back — round-trip
//     must be lossless.
//  3. Every tool must have a non-null inputSchema.
//  4. Every tool that exposes an outputSchema must have a non-null one.
//  5. Both schemas are checked recursively for empty {} fragments.
//  6. The tool count is asserted against the expected catalog size.
func TestToolsListClientView(t *testing.T) {
	type sessionCase struct {
		name          string
		deps          Deps
		wantToolCount int
	}

	cases := []sessionCase{
		{name: "base", deps: Deps{}, wantToolCount: 12},
		{name: "hooks-admin", deps: Deps{HooksAdminEnabled: true}, wantToolCount: 16},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session, ctx := mustNewMutationSession(t, tc.deps)
			defer session.Close()

			toolsRes, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatalf("ListTools() error = %v", err)
			}

			// 1. JSON round-trip
			raw, err := json.Marshal(toolsRes)
			if err != nil {
				t.Fatalf("marshal tools/list response: %v", err)
			}
			if !json.Valid(raw) {
				t.Fatalf("tools/list response is not valid JSON")
			}
			var roundTripped map[string]any
			if err := json.Unmarshal(raw, &roundTripped); err != nil {
				t.Fatalf("unmarshal tools/list response: %v", err)
			}

			// 2. Tool count
			if got := len(toolsRes.Tools); got != tc.wantToolCount {
				t.Errorf("tool count = %d want %d", got, tc.wantToolCount)
			}

			for _, tool := range toolsRes.Tools {
				toolName := tool.Name

				// 3. inputSchema must always be present
				if tool.InputSchema == nil {
					t.Errorf("tool %s: inputSchema is nil", toolName)
					continue
				}
				rawIn, err := json.Marshal(tool.InputSchema)
				if err != nil {
					t.Fatalf("tool %s: marshal inputSchema: %v", toolName, err)
				}
				if string(rawIn) == "null" {
					t.Errorf("tool %s: inputSchema serialises to null", toolName)
					continue
				}
				var nodeIn map[string]any
				if err := json.Unmarshal(rawIn, &nodeIn); err != nil {
					t.Fatalf("tool %s: decode inputSchema: %v", toolName, err)
				}
				for _, v := range collectEmptySchemaNodes(nodeIn, "inputSchema") {
					t.Errorf("tool %s: empty schema fragment at %s", toolName, v)
				}

				// 4 & 5. outputSchema — optional but must be valid when present
				if tool.OutputSchema == nil {
					continue
				}
				rawOut, err := json.Marshal(tool.OutputSchema)
				if err != nil {
					t.Fatalf("tool %s: marshal outputSchema: %v", toolName, err)
				}
				if string(rawOut) == "null" {
					t.Errorf("tool %s: outputSchema serialises to null", toolName)
					continue
				}
				var nodeOut map[string]any
				if err := json.Unmarshal(rawOut, &nodeOut); err != nil {
					t.Fatalf("tool %s: decode outputSchema: %v", toolName, err)
				}
				for _, v := range collectEmptySchemaNodes(nodeOut, "outputSchema") {
					t.Errorf("tool %s: empty schema fragment at %s", toolName, v)
				}
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
	res := mustCallToolResult(t, session, ctx, name, args)
	if res.StructuredContent != nil {
		raw, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatalf("CallTool(%s) structured content marshal error = %v", name, err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("CallTool(%s) structured content decode error = %v", name, err)
		}
		return got
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

func mustCallToolResult(t *testing.T, session *mcp.ClientSession, ctx context.Context, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}
	return res
}

func mustStructuredMap(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	if res.StructuredContent == nil {
		t.Fatal("missing structured content")
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode structured content: %v", err)
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
