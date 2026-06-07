package assets

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestListAssetsParity(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"))
	got, err := svc.List(context.Background(), ListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	gotNorm := normalizeAssets(got)
	want := ListResult{}
	mustLoadJSON(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle", "list_assets_all.normalized.json"), &want)
	want = normalizeAssets(want)
	if diff := compareJSON(t, gotNorm, want); diff != "" {
		t.Fatalf("List() mismatch:\n%s", diff)
	}
}

func TestListAssetsPathPrefixParity(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"))
	got, err := svc.List(context.Background(), ListRequest{Type: "image", PathPrefix: "posts/bonjour", MaxResults: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	gotNorm := normalizeAssets(got)
	want := ListResult{}
	mustLoadJSON(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle", "list_assets_post_images.normalized.json"), &want)
	want = normalizeAssets(want)
	if diff := compareJSON(t, gotNorm, want); diff != "" {
		t.Fatalf("List(path prefix) mismatch:\n%s", diff)
	}
}

func TestListAssetsPathPrefixSemantics(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"))
	tests := []struct {
		name       string
		pathPrefix string
		wantCount  int
		wantErr    string
	}{
		{name: "exact prefix", pathPrefix: "posts/bonjour", wantCount: 1},
		{name: "partial prefix", pathPrefix: "posts/bo", wantCount: 0},
		{name: "absent prefix", pathPrefix: "nope/nowhere", wantCount: 0},
		{name: "traversal rejected", pathPrefix: "../escape", wantErr: "Invalid path_prefix"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.List(context.Background(), ListRequest{Type: "image", PathPrefix: tc.pathPrefix, MaxResults: 20})
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("List() expected error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("List() error = %v want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if gotCount := len(got.Assets); gotCount != tc.wantCount {
				t.Fatalf("List() count = %d want %d", gotCount, tc.wantCount)
			}
		})
	}
}

func TestListAssetsTraversalRejected(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"))
	_, err := svc.List(context.Background(), ListRequest{PathPrefix: "../escape"})
	if err == nil {
		t.Fatal("List() expected error")
	}
}

func TestListAssetsCanceledContext(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.List(ctx, ListRequest{}); err == nil {
		t.Fatal("List() expected context error")
	}
}

func TestListAssetsIncludesUppercaseExtensions(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"))
	got, err := svc.List(context.Background(), ListRequest{Type: "image", PathPrefix: "posts/unicode", MaxResults: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	found := false
	for _, asset := range got.Assets {
		if asset.Path == "content/posts/unicode/COVER.SVG" {
			found = true
			if asset.MimeType != "image/svg+xml" {
				t.Fatalf("List() mime = %q", asset.MimeType)
			}
		}
	}
	if !found {
		t.Fatal("List() did not include uppercase COVER.SVG")
	}
}

func TestListAssetsPagination(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"))

	first, err := svc.List(context.Background(), ListRequest{MaxResults: 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(first.Assets) != 1 {
		t.Fatalf("List() len = %d want 1", len(first.Assets))
	}
	if first.NextCursor == "" {
		t.Fatal("List() expected next cursor")
	}
	if !first.HasMore {
		t.Fatal("List() expected HasMore=true")
	}

	second, err := svc.List(context.Background(), ListRequest{MaxResults: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("List(cursor) error = %v", err)
	}
	if len(second.Assets) == 0 {
		t.Fatal("List(cursor) returned no assets")
	}
	if second.Assets[0].Path == first.Assets[0].Path {
		t.Fatalf("List(cursor) returned duplicate asset %q", second.Assets[0].Path)
	}
}

func TestGetAssetChunk(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(staticDir, "big.bin")
	if err := os.WriteFile(path, []byte("0123456789abcdef0123456789abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(root, filepath.Join(root, "content"), staticDir)
	got, err := svc.GetChunk(context.Background(), ChunkRequest{Path: "static/big.bin", Cursor: 0, ChunkBytes: 8})
	if err != nil {
		t.Fatalf("GetChunk() error = %v", err)
	}
	if got.NextCursor == "" {
		t.Fatal("GetChunk() expected next cursor")
	}
	if got.IsLast {
		t.Fatal("GetChunk() expected more chunks")
	}
	if got.Chunk == "" {
		t.Fatal("GetChunk() returned empty chunk")
	}
}

func normalizeAssets(in ListResult) ListResult {
	out := in
	for i := range out.Assets {
		out.Assets[i].Modified = ""
	}
	sort.SliceStable(out.Assets, func(i, j int) bool {
		return out.Assets[i].Path < out.Assets[j].Path
	})
	return out
}

func mustLoadJSON(t *testing.T, path string, dst any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatal(err)
	}
}

func compareJSON(t *testing.T, got, want any) string {
	t.Helper()
	gb, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	wb, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(gb) == string(wb) {
		return ""
	}
	return "got:\n" + string(gb) + "\nwant:\n" + string(wb)
}
