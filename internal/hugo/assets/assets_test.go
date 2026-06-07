package assets

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
	if diff := compareJSON(t, gotNorm, want); diff != "" {
		t.Fatalf("List(path prefix) mismatch:\n%s", diff)
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

func normalizeAssets(in ListResult) ListResult {
	out := in
	for i := range out.Assets {
		out.Assets[i].Modified = ""
	}
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
