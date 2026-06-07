package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParityToolCatalogIncludesSRIAndFeaturedImage(t *testing.T) {
	session, ctx := mustNewMutationSession(t, newMutationDeps(t, &fakeBuildRunner{}))
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	got := make([]string, 0, len(toolsRes.Tools))
	for _, tool := range toolsRes.Tools {
		got = append(got, tool.Name)
		if tool.Description == "" {
			t.Fatalf("tool %q missing description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Fatalf("tool %q missing input schema", tool.Name)
		}
	}

	want := []string{
		"build_site",
		"check_sri_versions",
		"create_page",
		"delete_page",
		"generate_featured_image",
		"get_page",
		"list_assets",
		"list_pages",
		"update_page",
		"upload_asset",
	}
	if !equalStrings(got, want) {
		t.Fatalf("tool order = %#v want %#v", got, want)
	}
}

func TestCheckSRIVersionsParityContract(t *testing.T) {
	session, ctx := mustNewMutationSession(t, newMutationDeps(t, &fakeBuildRunner{}))
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "check_sri_versions",
		Arguments: map[string]any{
			"auto_fix": false,
			"dry_run":  true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(check_sri_versions) protocol error = %v", err)
	}
	if !res.IsError {
		// The contract is a structured success response when no audit handlers are installed.
		text := res.Content[0].(*mcp.TextContent).Text
		if text == "" {
			t.Fatal("CallTool(check_sri_versions) returned empty content")
		}
		return
	}
	t.Fatalf("CallTool(check_sri_versions) unexpectedly returned an error response: %#v", res)
}

func TestGenerateFeaturedImageParityContract(t *testing.T) {
	ws := newTestWorkspace(t)
	session, ctx := mustNewMutationSession(t, newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{}))
	defer session.Close()

	args := map[string]any{
		"style":    "tech",
		"title":    "Parity Tool",
		"subtitle": "featured image contract",
		"tags":     []string{"hugo", "mcp"},
		"accent":   "#7aa2f7",
		"slug":     "parity-tool",
		"route":    "/",
		"lang":     "fr",
	}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "generate_featured_image", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(generate_featured_image) protocol error = %v", err)
	}
	if res.IsError {
		t.Fatalf("CallTool(generate_featured_image) returned error: %#v", res)
	}

	got := map[string]any{}
	if err := json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &got); err != nil {
		t.Fatalf("CallTool(generate_featured_image) JSON decode error = %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("status = %v want ok", got["status"])
	}
	if got["filename"] != "parity-tool-featured.jpg" {
		t.Fatalf("filename = %v want parity-tool-featured.jpg", got["filename"])
	}
	if got["public_url"] != "/images/parity-tool-featured.jpg" {
		t.Fatalf("public_url = %v want /images/parity-tool-featured.jpg", got["public_url"])
	}
	if got["style"] != "tech" {
		t.Fatalf("style = %v want tech", got["style"])
	}
	if got["frontmatter_updated"] != true {
		t.Fatalf("frontmatter_updated = %v want true", got["frontmatter_updated"])
	}
	if len(got["langs_updated"].([]any)) == 0 {
		t.Fatal("langs_updated unexpectedly empty")
	}
	if _, err := os.Stat(filepath.Join(ws.StaticRoot, "images", "parity-tool-featured.jpg")); err != nil {
		t.Fatalf("generated image missing: %v", err)
	}
}

func TestParityToolsRejectInvalidArguments(t *testing.T) {
	session, ctx := mustNewMutationSession(t, newMutationDeps(t, &fakeBuildRunner{}))
	defer session.Close()

	cases := []struct {
		name string
		tool string
		args map[string]any
	}{
		{
			name: "check sri invalid type",
			tool: "check_sri_versions",
			args: map[string]any{"auto_fix": "yes"},
		},
		{
			name: "featured image bad slug",
			tool: "generate_featured_image",
			args: map[string]any{
				"title": "Parity Tool",
				"slug":  "../escape",
			},
		},
		{
			name: "featured image bad accent",
			tool: "generate_featured_image",
			args: map[string]any{
				"title":  "Parity Tool",
				"slug":   "parity-tool",
				"accent": "not-a-color",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err == nil && (res == nil || !res.IsError) {
				t.Fatalf("CallTool(%s) unexpectedly succeeded: %#v", tc.tool, res)
			}
		})
	}
}

func TestGenerateFeaturedImageMissingPageFailsClosed(t *testing.T) {
	session, ctx := mustNewMutationSession(t, newMutationDeps(t, &fakeBuildRunner{}))
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "generate_featured_image",
		Arguments: map[string]any{
			"title": "Parity Tool",
			"slug":  "parity-tool",
			"route": "/posts/does-not-exist",
			"lang":  "fr",
		},
	})
	if err == nil && (res == nil || !res.IsError) {
		t.Fatalf("CallTool(generate_featured_image) unexpectedly succeeded: %#v", res)
	}
}
