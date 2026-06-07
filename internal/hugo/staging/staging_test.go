package staging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewWorkspaceAcceptsCanonicalRoots(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	static := filepath.Join(root, "static")
	public := filepath.Join(root, "public")
	work := filepath.Join(root, "work")
	for _, dir := range []string{content, static, public, work} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := New(root, content, static, public, work)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if ws.HugoRoot == "" || ws.ContentRoot == "" || ws.StaticRoot == "" || ws.PublicRoot == "" || ws.WorkRoot == "" {
		t.Fatalf("New() returned empty workspace: %#v", ws)
	}
}

func TestNewWorkspaceRejectsMissingRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := New(root, filepath.Join(root, "content"), filepath.Join(root, "static"), filepath.Join(root, "public"), filepath.Join(root, "work")); err == nil {
		t.Fatal("New() expected missing-root error")
	}
}

func TestNewWorkspaceRejectsSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	content := filepath.Join(root, "content")
	if err := os.Symlink(real, content); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"static", "public", "work"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := New(root, content, filepath.Join(root, "static"), filepath.Join(root, "public"), filepath.Join(root, "work")); err == nil {
		t.Fatal("New() expected symlink-root error")
	}
}

func TestResolveNewContentRejectsOverwriteAndEscape(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	static := filepath.Join(root, "static")
	public := filepath.Join(root, "public")
	work := filepath.Join(root, "work")
	for _, dir := range []string{content, static, public, work} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := New(root, content, static, public, work)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(content, "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(content, "posts", "existing.md")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ResolveNewContent("posts/existing.md"); err == nil {
		t.Fatal("ResolveNewContent() expected overwrite error")
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(content, "posts", "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ResolveNewContent("posts/escape/new.md"); err == nil {
		t.Fatal("ResolveNewContent() expected symlink escape error")
	}
}

func TestResolveNewPublicAndWork(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	static := filepath.Join(root, "static")
	public := filepath.Join(root, "public")
	work := filepath.Join(root, "work")
	for _, dir := range []string{content, static, public, work} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := New(root, content, static, public, work)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ResolveNewPublic("build/index.html"); err != nil {
		t.Fatalf("ResolveNewPublic() error = %v", err)
	}
	if _, err := ws.ResolveNewWork("delete/page.md"); err != nil {
		t.Fatalf("ResolveNewWork() error = %v", err)
	}
}
