package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRoots(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	static := filepath.Join(root, "static")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(static, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := FromRoots(root, content, static)
	got, err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got.HugoRoot == "" || got.ContentRoot == "" || got.StaticRoot == "" {
		t.Fatalf("Validate() returned empty roots: %#v", got)
	}
}

func TestValidateMissingRoots(t *testing.T) {
	_, err := (Config{}).Validate()
	if err == nil {
		t.Fatal("Validate() expected error")
	}
}

func TestLoadFromEnvRejectsInvalidNumbers(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HUGO_ROOT", root)
	t.Setenv("HUGO_CONTENT_ROOT", filepath.Join(root, "content"))
	t.Setenv("HUGO_STATIC_ROOT", filepath.Join(root, "static"))
	t.Setenv("HUGO_MAX_LIST_PAGES", "abc")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("LoadFromEnv() expected error")
	}
}

func TestValidateRejectsContentOutsideRoot(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(t.TempDir(), "content-outside")
	static := filepath.Join(root, "static")
	if err := os.MkdirAll(static, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := FromRoots(root, content, static)
	if _, err := cfg.Validate(); err == nil {
		t.Fatal("Validate() expected error for content outside root")
	}
}
