package server

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerExposesReadOnlyTools(t *testing.T) {
	cfg := config.FromRoots(
		filepath.Join("..", "..", "testdata", "fixtures", "minimal-site"),
		filepath.Join("..", "..", "testdata", "fixtures", "minimal-site", "content"),
		filepath.Join("..", "..", "testdata", "fixtures", "minimal-site", "static"),
	)
	svc, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := svc.MCP().Connect(ctx, t1, nil); err != nil {
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
	gotNames := map[string]bool{}
	for _, tool := range tools.Tools {
		gotNames[tool.Name] = true
	}
	for _, name := range []string{"list_pages", "get_page", "list_assets"} {
		if !gotNames[name] {
			t.Fatalf("tool %q not exposed", name)
		}
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_pages", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("CallTool(list_pages) error = %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("CallTool(list_pages) returned no content")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("list_pages JSON decode error = %v", err)
	}
}
