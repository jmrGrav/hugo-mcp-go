package pathguard

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func CanonicalDir(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("empty root")
	}
	if strings.Contains(root, "\\") {
		return "", fmt.Errorf("root contains backslashes")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root is not a directory")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("root must not be a symlink")
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	if filepath.Clean(eval) != filepath.Clean(abs) {
		return "", fmt.Errorf("root must not resolve through a symlink")
	}
	return abs, nil
}

func ValidateRelative(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", nil
	}
	if strings.Contains(s, "\\") {
		return "", fmt.Errorf("path contains backslashes")
	}
	if strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("path must be relative")
	}
	if strings.Contains(s, "..") {
		return "", fmt.Errorf("path traversal detected")
	}
	clean := path.Clean(s)
	if clean == "." {
		return "", nil
	}
	return clean, nil
}

func WithinRoot(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	if root == candidate {
		return true
	}
	if !strings.HasSuffix(root, string(os.PathSeparator)) {
		root += string(os.PathSeparator)
	}
	return strings.HasPrefix(candidate, root)
}

func ResolveExistingPath(root, rel string) (string, error) {
	canonicalRoot, err := CanonicalDir(root)
	if err != nil {
		return "", err
	}
	cleanRel, err := ValidateRelative(rel)
	if err != nil {
		return "", err
	}
	candidate := canonicalRoot
	if cleanRel != "" {
		candidate = filepath.Join(canonicalRoot, filepath.FromSlash(cleanRel))
	}
	info, err := os.Lstat(candidate)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlinks are not allowed")
	}
	eval, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	if !WithinRoot(canonicalRoot, eval) {
		return "", fmt.Errorf("resolved path escapes root")
	}
	if filepath.Clean(eval) != filepath.Clean(candidate) {
		return "", fmt.Errorf("symlinks are not allowed")
	}
	return candidate, nil
}

func ResolveScanRoot(root, rel string) (string, error) {
	if rel == "" {
		return CanonicalDir(root)
	}
	return ResolveExistingPath(root, rel)
}

func WalkDirNoSymlink(root string, fn func(string, fs.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed")
		}
		return fn(path, d)
	})
}
