package pages

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListPagesParity(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	got, err := svc.List(context.Background(), ListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := ListResult{}
	mustLoadJSON(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle", "list_pages_all.normalized.json"), &want)
	if diff := compareJSON(t, got, want); diff != "" {
		t.Fatalf("List() mismatch:\n%s", diff)
	}
}

func TestListPagesSectionLangParity(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	got, err := svc.List(context.Background(), ListRequest{Lang: "fr", Section: "posts"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := ListResult{}
	mustLoadJSON(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle", "list_pages_posts_fr.normalized.json"), &want)
	if diff := compareJSON(t, got, want); diff != "" {
		t.Fatalf("List(section/lang) mismatch:\n%s", diff)
	}
}

func TestGetPageParity(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	got, err := svc.Get(context.Background(), GetRequest{Route: "/posts/bonjour", Lang: "fr"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	want := Page{}
	mustLoadJSON(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle", "get_page_bonjour_fr.normalized.json"), &want)
	if diff := compareJSON(t, got, want); diff != "" {
		t.Fatalf("Get() mismatch:\n%s", diff)
	}
}

func TestGetRootFallbackParity(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	got, err := svc.Get(context.Background(), GetRequest{Route: "_index", Lang: "fr"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	want := Page{}
	mustLoadJSON(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle", "get_page_root_fr.normalized.json"), &want)
	if diff := compareJSON(t, got, want); diff != "" {
		t.Fatalf("Get(root) mismatch:\n%s", diff)
	}
}

func TestGetPageMissing(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	_, err := svc.Get(context.Background(), GetRequest{Route: "/posts/missing", Lang: "fr"})
	if err == nil {
		t.Fatal("Get() expected error")
	}
	if got, want := err.Error(), "Page not found: posts/missing (lang=fr)"; got != want {
		t.Fatalf("Get() error = %q want %q", got, want)
	}
}

func TestGetPageFallsBackToPlainIndexMarkdown(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	got, err := svc.Get(context.Background(), GetRequest{Route: "/posts/plain", Lang: "zz"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.File != "posts/plain/index.md" {
		t.Fatalf("Get() file = %q", got.File)
	}
	if got.Route != "posts/plain" {
		t.Fatalf("Get() route = %q", got.Route)
	}
	if got.Frontmatter["title"] != nil {
		t.Fatalf("Get() frontmatter title = %#v", got.Frontmatter["title"])
	}
	if got.Content == "" || got.Content[:10] != "This page " {
		t.Fatalf("Get() content = %q", got.Content)
	}
}

func TestListPagesIgnoresPlainIndexMarkdown(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	got, err := svc.List(context.Background(), ListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	for _, page := range got.Pages {
		if page.File == "posts/plain/index.md" {
			t.Fatalf("List() unexpectedly included plain index.md: %#v", page)
		}
	}
}

func TestListPagesTraversalRejected(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	_, err := svc.List(context.Background(), ListRequest{Section: "../escape"})
	if err == nil {
		t.Fatal("List() expected error")
	}
}

func TestListPagesRejectsSymlinkSection(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(filepath.Join(content, "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outside, "escape"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "escape", "index.fr.md"), []byte("---\ntitle: leaked\n---\npayload\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "escape"), filepath.Join(content, "posts", "escape")); err != nil {
		t.Fatal(err)
	}
	svc := New(content)
	_, err := svc.List(context.Background(), ListRequest{Section: "posts/escape"})
	if err == nil {
		t.Fatal("List() expected symlink error")
	}
}

func TestGetPageRejectsOversizedFile(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	if err := os.MkdirAll(filepath.Join(content, "posts", "big"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 0, 128)
	raw = append(raw, []byte("---\ntitle: big\n---\n")...)
	raw = append(raw, []byte(strings.Repeat("x", 64))...)
	if err := os.WriteFile(filepath.Join(content, "posts", "big", "index.fr.md"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(content)
	svc.MaxPageBytes = 32
	_, err := svc.Get(context.Background(), GetRequest{Route: "/posts/big", Lang: "fr"})
	if err == nil {
		t.Fatal("Get() expected oversized file error")
	}
}

func TestListPagesPagination(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))

	first, err := svc.List(context.Background(), ListRequest{Limit: 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(first.Pages) != 1 {
		t.Fatalf("List() len = %d want 1", len(first.Pages))
	}
	if first.NextCursor == "" {
		t.Fatal("List() expected next cursor")
	}
	if !first.HasMore {
		t.Fatal("List() expected HasMore=true")
	}

	second, err := svc.List(context.Background(), ListRequest{Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("List(cursor) error = %v", err)
	}
	if len(second.Pages) == 0 {
		t.Fatal("List(cursor) returned no pages")
	}
	if second.Pages[0].File == first.Pages[0].File {
		t.Fatalf("List(cursor) returned duplicate page %q", second.Pages[0].File)
	}
}

func TestGetPageChunkSuggestsChunking(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	if err := os.MkdirAll(filepath.Join(content, "posts", "big"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 0, 128)
	raw = append(raw, []byte("---\ntitle: big\n---\n")...)
	raw = append(raw, []byte(strings.Repeat("x", 64))...)
	if err := os.WriteFile(filepath.Join(content, "posts", "big", "index.fr.md"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(content)
	svc.MaxPageBytes = 32
	_, err := svc.Get(context.Background(), GetRequest{Route: "/posts/big", Lang: "fr"})
	if err == nil {
		t.Fatal("Get() expected oversized file error")
	}
	if !strings.Contains(err.Error(), "get_page_chunk") {
		t.Fatalf("Get() error = %q want chunk suggestion", err.Error())
	}
}

func TestListPagesCanceledContext(t *testing.T) {
	svc := New(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.List(ctx, ListRequest{}); err == nil {
		t.Fatal("List() expected context error")
	}
}

func TestListPagesSkipsUnreadableFile(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	if err := os.MkdirAll(filepath.Join(content, "posts", "skipme"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "posts", "skipme", "index.fr.md"), []byte("not readable"), 0o000); err != nil {
		t.Fatal(err)
	}
	svc := New(content)
	got, err := svc.List(context.Background(), ListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got.Skipped == nil || *got.Skipped == 0 {
		t.Fatalf("List() skipped count not set: %#v", got)
	}
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
