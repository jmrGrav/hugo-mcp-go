package pathguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveNewTargetPathAcceptsNormalPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "posts", "bonjour"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveNewTargetPath(root, "posts/bonjour/new.fr.md")
	if err != nil {
		t.Fatalf("ResolveNewTargetPath() error = %v", err)
	}
	want := filepath.Join(root, "posts", "bonjour", "new.fr.md")
	if got != want {
		t.Fatalf("ResolveNewTargetPath() = %q want %q", got, want)
	}
}

func TestResolveNewTargetPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../escape.md", "/abs/path.md", `posts\escape.md`, "posts/../escape.md"} {
		if _, err := ResolveNewTargetPath(root, rel); err == nil {
			t.Fatalf("ResolveNewTargetPath(%q) expected error", rel)
		}
	}
}

func TestResolveNewTargetPathRejectsMissingRoot(t *testing.T) {
	if _, err := ResolveNewTargetPath(filepath.Join(t.TempDir(), "missing"), "posts/new.md"); err == nil {
		t.Fatal("ResolveNewTargetPath() expected error for missing root")
	}
}

func TestResolveNewTargetPathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content", "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "content", "posts", "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveNewTargetPath(filepath.Join(root, "content"), "posts/escape/new.md"); err == nil {
		t.Fatal("ResolveNewTargetPath() expected symlink escape error")
	}
}

func TestResolveNewTargetPathRejectsOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "posts", "existing.md")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveNewTargetPath(root, "posts/existing.md"); err == nil {
		t.Fatal("ResolveNewTargetPath() expected overwrite error")
	}
}
