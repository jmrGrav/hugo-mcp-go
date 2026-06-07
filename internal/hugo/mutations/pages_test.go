package mutations

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/frontmatter"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
)

func TestCreatePageNominal(t *testing.T) {
	svc := newTestPageService(t)
	svc.Now = fixedMutationTime

	req := CreatePageRequest{
		Route:   "/posts/oracle-phase2",
		Lang:    "fr",
		Title:   "Phase 2 oracle",
		Content: "Corps de page créé pour la capture oracle.",
		Tags:    []string{"oracle", "phase2"},
		Draft:   boolPtr(false),
		Frontmatter: map[string]any{
			"description": "Oracle nominal create",
			"date":        "2026-06-06T10:00:00+02:00",
			"lastmod":     "2026-06-06T10:05:00+02:00",
			"seo": map[string]any{
				"summary": "Create oracle",
			},
		},
	}

	got, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	assertMutationResult(t, got, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "create_page.response.json"))
	assertPageSnapshot(t, filepath.Join(svc.Stage.ContentRoot, "posts", "oracle-phase2", "index.fr.md"), filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "create_page.after", "content", "posts", "oracle-phase2", "index.fr.md.json"))
}

func TestCreatePageRejectsFrontmatterConflict(t *testing.T) {
	svc := newTestPageService(t)
	_, err := svc.Create(context.Background(), CreatePageRequest{
		Route:   "/posts/conflict",
		Lang:    "fr",
		Title:   "Conflict title",
		Content: "Body",
		Frontmatter: map[string]any{
			"title": "Illegal duplicate",
		},
	})
	if err == nil {
		t.Fatal("Create() expected error")
	}
	if got, want := err.Error(), "Conflict: field(s) provided both as dedicated param and in frontmatter: title. Use only one."; got != want {
		t.Fatalf("Create() error = %q want %q", got, want)
	}
}

func TestCreatePageTraversalRejected(t *testing.T) {
	svc := newTestPageService(t)
	_, err := svc.Create(context.Background(), CreatePageRequest{
		Route:   "../escape",
		Lang:    "fr",
		Title:   "x",
		Content: "x",
	})
	if err == nil {
		t.Fatal("Create() expected error")
	}
}

func TestCreatePageSymlinkRejected(t *testing.T) {
	svc := newTestPageService(t)
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(svc.Stage.ContentRoot, "posts", "escape")); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(context.Background(), CreatePageRequest{
		Route:   "/posts/escape/new",
		Lang:    "fr",
		Title:   "x",
		Content: "x",
	})
	if err == nil {
		t.Fatal("Create() expected symlink error")
	}
}

func TestCreatePageOverwriteRejected(t *testing.T) {
	svc := newTestPageService(t)
	_, err := svc.Create(context.Background(), CreatePageRequest{
		Route:   "/posts/bonjour",
		Lang:    "fr",
		Title:   "Bonjour",
		Content: "x",
	})
	if err == nil {
		t.Fatal("Create() expected overwrite error")
	}
}

func TestCreatePageRejectsOversizedContent(t *testing.T) {
	svc := newTestPageService(t)
	svc.MaxPageBytes = 64
	big := strings.Repeat("x", 128)
	_, err := svc.Create(context.Background(), CreatePageRequest{
		Route:   "/posts/too-big",
		Lang:    "fr",
		Title:   "x",
		Content: big,
	})
	if err == nil {
		t.Fatal("Create() expected oversized content error")
	}
	if got, want := err.Error(), "page too large"; !strings.Contains(got, want) {
		t.Fatalf("Create() error = %q want substring %q", got, want)
	}
}

func TestUpdatePageNominal(t *testing.T) {
	svc := newTestPageService(t)
	svc.Now = fixedMutationTime

	title := "Bonjour mis à jour"
	content := "Contenu français mis à jour."
	tags := []string{"posts", "fr", "oracle"}
	req := UpdatePageRequest{
		Route:   "/posts/bonjour",
		Lang:    "fr",
		Title:   &title,
		Content: &content,
		Tags:    &tags,
		Draft:   boolPtr(false),
		Frontmatter: map[string]any{
			"description": nil,
			"seo": map[string]any{
				"summary": "Updated",
			},
			"lastmod": "2026-06-06T11:00:00+02:00",
		},
	}

	got, err := svc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	assertMutationResult(t, got, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "update_page.response.json"))
	assertPageSnapshot(t, filepath.Join(svc.Stage.ContentRoot, "posts", "bonjour", "index.fr.md"), filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "update_page.after", "content", "posts", "bonjour", "index.fr.md.json"))
}

func TestUpdatePageRejectsImmutableDate(t *testing.T) {
	svc := newTestPageService(t)
	_, err := svc.Update(context.Background(), UpdatePageRequest{
		Route: "/posts/bonjour",
		Lang:  "fr",
		Frontmatter: map[string]any{
			"date": "2020-01-01T00:00:00+00:00",
		},
	})
	if err == nil {
		t.Fatal("Update() expected error")
	}
	if got, want := err.Error(), "Field(s) cannot be modified via update_page: date"; got != want {
		t.Fatalf("Update() error = %q want %q", got, want)
	}
}

func TestUpdatePageMissing(t *testing.T) {
	svc := newTestPageService(t)
	_, err := svc.Update(context.Background(), UpdatePageRequest{
		Route: "/posts/missing",
		Lang:  "fr",
	})
	if err == nil {
		t.Fatal("Update() expected error")
	}
	if got, want := err.Error(), "Page not found: posts/missing"; got != want {
		t.Fatalf("Update() error = %q want %q", got, want)
	}
}

func TestUpdatePageSymlinkRejected(t *testing.T) {
	svc := newTestPageService(t)
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(svc.Stage.ContentRoot, "posts", "bonjour", "index.fr.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "page.md"), filepath.Join(svc.Stage.ContentRoot, "posts", "bonjour", "index.fr.md")); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Update(context.Background(), UpdatePageRequest{
		Route: "/posts/bonjour",
		Lang:  "fr",
	})
	if err == nil {
		t.Fatal("Update() expected symlink error")
	}
}

func TestUpdatePageRejectsOversizedContent(t *testing.T) {
	svc := newTestPageService(t)
	svc.MaxPageBytes = 64
	big := strings.Repeat("x", 128)
	_, err := svc.Update(context.Background(), UpdatePageRequest{
		Route:   "/posts/bonjour",
		Lang:    "fr",
		Content: &big,
	})
	if err == nil {
		t.Fatal("Update() expected oversized content error")
	}
	if got, want := err.Error(), "page too large"; !strings.Contains(got, want) {
		t.Fatalf("Update() error = %q want substring %q", got, want)
	}
}

func TestDeletePageNominal(t *testing.T) {
	svc := newTestPageService(t)
	got, err := svc.Delete(context.Background(), DeletePageRequest{Route: "/posts/plain"})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertMutationResult(t, got, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "delete_page.response.json"))
	if _, err := os.Stat(filepath.Join(svc.Stage.ContentRoot, "posts", "plain", "index.md")); !os.IsNotExist(err) {
		t.Fatalf("Delete() file still exists: %v", err)
	}
	entries, err := os.ReadDir(svc.Stage.WorkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("Delete() did not leave rollback breadcrumb")
	}
}

func TestDeletePageMissing(t *testing.T) {
	svc := newTestPageService(t)
	_, err := svc.Delete(context.Background(), DeletePageRequest{Route: "/posts/missing", Lang: "fr"})
	if err == nil {
		t.Fatal("Delete() expected error")
	}
	if got, want := err.Error(), "Page not found: posts/missing"; got != want {
		t.Fatalf("Delete() error = %q want %q", got, want)
	}
}

func TestDeletePageTraversalRejected(t *testing.T) {
	svc := newTestPageService(t)
	_, err := svc.Delete(context.Background(), DeletePageRequest{Route: "../escape"})
	if err == nil {
		t.Fatal("Delete() expected error")
	}
}

func TestDeletePageSymlinkRejected(t *testing.T) {
	svc := newTestPageService(t)
	if err := os.Remove(filepath.Join(svc.Stage.ContentRoot, "posts", "plain", "index.md")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "page.md"), filepath.Join(svc.Stage.ContentRoot, "posts", "plain", "index.md")); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Delete(context.Background(), DeletePageRequest{Route: "/posts/plain"})
	if err == nil {
		t.Fatal("Delete() expected symlink error")
	}
}

func TestUpdatePageIdempotentSemantic(t *testing.T) {
	svc := newTestPageService(t)
	svc.Now = fixedMutationTime
	title := "Bonjour mis à jour"
	content := "Contenu français mis à jour."
	tags := []string{"posts", "fr", "oracle"}
	req := UpdatePageRequest{
		Route:   "/posts/bonjour",
		Lang:    "fr",
		Title:   &title,
		Content: &content,
		Tags:    &tags,
		Draft:   boolPtr(false),
		Frontmatter: map[string]any{
			"lastmod": "2026-06-06T11:00:00+02:00",
		},
	}
	if _, err := svc.Update(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	first := readPageSnapshot(t, filepath.Join(svc.Stage.ContentRoot, "posts", "bonjour", "index.fr.md"))
	if _, err := svc.Update(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	second := readPageSnapshot(t, filepath.Join(svc.Stage.ContentRoot, "posts", "bonjour", "index.fr.md"))
	if !equalJSON(t, first, second) {
		t.Fatal("Update() is not semantically idempotent")
	}
}

func newTestPageService(t *testing.T) *PageService {
	t.Helper()
	ws := newTestWorkspace(t)
	return NewPageService(ws)
}

func newTestWorkspace(t *testing.T) *staging.Workspace {
	t.Helper()
	root := t.TempDir()
	content := filepath.Join(root, "content")
	static := filepath.Join(root, "static")
	public := filepath.Join(root, "public")
	work := filepath.Join(root, "work")
	copyDir(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content"), content)
	copyDir(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "static"), static)
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

func assertMutationResult(t *testing.T, got MutationResult, snapshotPath string) {
	t.Helper()
	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want.Content) != 1 {
		t.Fatalf("unexpected snapshot content: %#v", want)
	}
	var wantResult map[string]any
	if err := json.Unmarshal([]byte(want.Content[0].Text), &wantResult); err != nil {
		t.Fatal(err)
	}
	gotRaw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var gotResult map[string]any
	if err := json.Unmarshal(gotRaw, &gotResult); err != nil {
		t.Fatal(err)
	}
	if !equalJSON(t, gotResult, wantResult) {
		t.Fatalf("mutation result mismatch:\n got=%s\nwant=%s", string(gotRaw), want.Content[0].Text)
	}
}

func assertPageSnapshot(t *testing.T, gotPath, snapshotPath string) {
	t.Helper()
	gotRaw, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatal(err)
	}
	wantRaw, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	gotFM, gotBody, err := frontmatter.Split(gotRaw)
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(wantRaw, &want); err != nil {
		t.Fatal(err)
	}
	wantFM, wantBody, err := frontmatter.Split([]byte(want.Content))
	if err != nil {
		t.Fatal(err)
	}
	if !equalJSON(t, gotFM, wantFM) {
		t.Fatalf("frontmatter mismatch:\n got=%v\nwant=%v", gotFM, wantFM)
	}
	if gotBody != wantBody {
		t.Fatalf("body mismatch:\n got=%q\nwant=%q", gotBody, wantBody)
	}
}

func readPageSnapshot(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm, content, err := frontmatter.Split(raw)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"frontmatter": fm,
		"content":     content,
	}
}

func equalJSON(t *testing.T, a, b any) bool {
	t.Helper()
	ab, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	bb, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	return string(ab) == string(bb)
}

func fixedMutationTime() time.Time {
	return time.Date(2026, 6, 6, 10, 0, 0, 0, time.FixedZone("+02", 2*3600))
}

func boolPtr(v bool) *bool { return &v }
