package pathguard

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalDirRejectsMissing(t *testing.T) {
	if _, err := CanonicalDir(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("CanonicalDir() expected error for missing root")
	}
}

func TestValidateRelative(t *testing.T) {
	got, err := ValidateRelative("posts/bonjour/index.fr.md")
	if err != nil {
		t.Fatalf("ValidateRelative() error = %v", err)
	}
	if got != "posts/bonjour/index.fr.md" {
		t.Fatalf("ValidateRelative() = %q", got)
	}
}

func TestValidateRelativeRejectsTraversal(t *testing.T) {
	for _, input := range []string{"../escape", "/abs/path", "posts\\bonjour", "posts/../escape"} {
		if _, err := ValidateRelative(input); err == nil {
			t.Fatalf("ValidateRelative(%q) expected error", input)
		}
	}
}

func TestResolveExistingPathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "content")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "page.md"), []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(allowed, "escape")); err != nil {
		t.Fatal(err)
	}

	if _, err := ResolveExistingPath(allowed, "escape/page.md"); err == nil {
		t.Fatal("ResolveExistingPath() expected symlink escape error")
	}
}

func TestResolveExistingPathAcceptsNormalPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "posts", "bonjour"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "posts", "bonjour", "index.fr.md")
	if err := os.WriteFile(file, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveExistingPath(root, "posts/bonjour/index.fr.md")
	if err != nil {
		t.Fatalf("ResolveExistingPath() error = %v", err)
	}
	if got != file {
		t.Fatalf("ResolveExistingPath() = %q want %q", got, file)
	}
}

func TestOpenDirChainCreatesMissingChain(t *testing.T) {
	root := t.TempDir()
	dir, err := OpenDirChain(root, "posts/bonjour", true)
	if err != nil {
		t.Fatalf("OpenDirChain() error = %v", err)
	}
	defer dir.Close()
	if _, err := os.Stat(filepath.Join(root, "posts", "bonjour")); err != nil {
		t.Fatalf("OpenDirChain() did not create directories: %v", err)
	}
}

func TestOpenExistingFileRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "posts", "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenExistingFile(root, "posts/escape/page.md"); err == nil {
		t.Fatal("OpenExistingFile() expected symlink error")
	}
}

func TestCreateTempFileRenameAndUnlink(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir, err := OpenDirChain(root, "files", false)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	name, f, err := CreateTempFile(dir, ".tmp-", 0o644)
	if err != nil {
		t.Fatalf("CreateTempFile() error = %v", err)
	}
	if _, err := io.WriteString(f, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := RenameInDir(dir, name, "final.txt", true); err != nil {
		t.Fatalf("RenameInDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "files", "final.txt")); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
	if err := UnlinkInDir(dir, "final.txt"); err != nil {
		t.Fatalf("UnlinkInDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "files", "final.txt")); !os.IsNotExist(err) {
		t.Fatalf("UnlinkInDir() did not remove file: %v", err)
	}
}
