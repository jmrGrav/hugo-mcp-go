package tools

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/sri"
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
		"get_asset_chunk",
		"get_page",
		"get_page_chunk",
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
	deps := newSriParityDeps(t)
	session, ctx := mustNewMutationSession(t, deps)
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
	if res.IsError {
		t.Fatalf("CallTool(check_sri_versions) returned error: %#v", res)
	}
	got := decodeToolJSON(t, res)
	if got["plugin"] != "sri-check" {
		t.Fatalf("plugin = %v want sri-check", got["plugin"])
	}
	if got["success"] != true {
		t.Fatalf("success = %v want true", got["success"])
	}
	if got["exit_code"] != float64(0) {
		t.Fatalf("exit_code = %v want 0", got["exit_code"])
	}
	report := got["report"].(map[string]any)
	if report["summary"] != "OK" {
		t.Fatalf("report.summary = %v want OK", report["summary"])
	}
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

func newSriParityDeps(t *testing.T) Deps {
	t.Helper()
	ws := newTestWorkspace(t)
	mustWrite(t, filepath.Join(ws.HugoRoot, "themes", "LoveIt", "assets", "data", "cdn"), "jsdelivr.yml", []byte(`prefix:
  libFiles: https://cdn.jsdelivr.net/npm/
libFiles:
  animateCSS: animate.css@4.1.1/animate.min.css
`))
	mustWrite(t, filepath.Join(ws.HugoRoot, "data"), "sri.yaml", []byte(`"https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css": "sha256-`+hashForTool([]byte("animate-ok"))+`"
`))
	mustWrite(t, filepath.Join(ws.HugoRoot, "public"), "index.html", []byte(`<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`))
	mustWrite(t, filepath.Join(ws.HugoRoot, "themes", "LoveIt", "layouts", "_partials"), "head.html", []byte(`{{/* https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css */}}`))

	svc, err := sri.NewService(sri.Config{
		Enabled:           true,
		TriggerHooksOnFix: true,
		DryRunDefault:     true,
		MaxFileBytes:      1 << 20,
		MaxFiles:          256,
		HugoRoot:          ws.HugoRoot,
		SiteBaseURL:       "https://example.com",
		AllowedCDNHosts:   []string{"cdn.jsdelivr.net", "fastly.jsdelivr.net", "data.jsdelivr.com"},
		ScanRoots:         []string{"public", "themes/LoveIt/layouts", "content"},
	}, sri.WithHTTPClient(&http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "data.jsdelivr.com":
			return jsonResponse(`{"tags":{"latest":"4.1.1"}}`), nil
		case req.URL.String() == "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css":
			return bodyResponse("animate-ok"), nil
		default:
			return bodyResponse("animate-ok"), nil
		}
	})}))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	deps := newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{})
	deps.Sri = svc
	return deps
}

func hashForTool(body []byte) string {
	sum := sha256.Sum256(body)
	return base64.StdEncoding.EncodeToString(sum[:])
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mustWrite(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s/%s) error = %v", dir, name, err)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func bodyResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
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
