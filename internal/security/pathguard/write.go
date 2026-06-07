package pathguard

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveNewTargetPath validates a not-yet-existing target under root.
// It rejects traversal, absolute user paths, symlink escapes, and existing targets.
func ResolveNewTargetPath(root, rel string) (string, error) {
	canonicalRoot, err := CanonicalDir(root)
	if err != nil {
		return "", err
	}
	cleanRel, err := ValidateRelative(rel)
	if err != nil {
		return "", err
	}
	if cleanRel == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	candidate := filepath.Join(canonicalRoot, filepath.FromSlash(cleanRel))
	if _, err := os.Lstat(candidate); err == nil {
		return "", fmt.Errorf("target already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(candidate)
	if _, err := safeExistingAncestor(canonicalRoot, parent); err != nil {
		return "", err
	}
	return candidate, nil
}

func safeExistingAncestor(root, dir string) (string, error) {
	root = filepath.Clean(root)
	dir = filepath.Clean(dir)
	if dir == root {
		return root, nil
	}
	info, err := os.Lstat(dir)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlinks are not allowed")
		}
		if !info.IsDir() {
			return "", fmt.Errorf("path parent is not a directory")
		}
		eval, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return "", err
		}
		if !WithinRoot(root, eval) {
			return "", fmt.Errorf("resolved path escapes root")
		}
		if filepath.Clean(eval) != filepath.Clean(dir) {
			return "", fmt.Errorf("symlinks are not allowed")
		}
		return dir, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(dir)
	if parent == dir {
		return "", fmt.Errorf("parent directory missing")
	}
	return safeExistingAncestor(root, parent)
}
