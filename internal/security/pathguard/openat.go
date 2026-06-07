package pathguard

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func OpenDirChain(root, rel string, createMissing bool) (*os.File, error) {
	canonicalRoot, err := CanonicalDir(root)
	if err != nil {
		return nil, err
	}
	rootFD, err := openNoFollowDir(canonicalRoot)
	if err != nil {
		return nil, err
	}
	parts, err := normalizeChain(rel)
	if err != nil {
		rootFD.Close()
		return nil, err
	}
	current := rootFD
	for _, part := range parts {
		next, err := openChildDir(current, part)
		if err == nil {
			_ = current.Close()
			current = next
			continue
		}
		if !createMissing || !errors.Is(err, unix.ENOENT) {
			_ = current.Close()
			return nil, err
		}
		if err := unix.Mkdirat(int(current.Fd()), part, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
			_ = current.Close()
			return nil, err
		}
		next, err = openChildDir(current, part)
		if err != nil {
			_ = current.Close()
			return nil, err
		}
		_ = current.Close()
		current = next
	}
	return current, nil
}

func OpenExistingFile(root, rel string) (*os.File, error) {
	cleanRel, err := ValidateRelative(rel)
	if err != nil {
		return nil, err
	}
	dirRel := filepath.ToSlash(filepath.Dir(cleanRel))
	base := filepath.Base(cleanRel)
	dir, err := OpenDirChain(root, dirRel, false)
	if err != nil {
		return nil, err
	}
	fd, err := unix.Openat(int(dir.Fd()), base, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = dir.Close()
		return nil, err
	}
	_ = dir.Close()
	return os.NewFile(uintptr(fd), filepath.Join(dir.Name(), base)), nil
}

func CreateTempFile(dir *os.File, prefix string, mode os.FileMode) (string, *os.File, error) {
	if dir == nil {
		return "", nil, fmt.Errorf("missing directory")
	}
	for i := 0; i < 256; i++ {
		suffix, err := randomSuffix()
		if err != nil {
			return "", nil, err
		}
		name := prefix + suffix
		fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(mode.Perm()))
		if err == nil {
			return name, os.NewFile(uintptr(fd), filepath.Join(dir.Name(), name)), nil
		}
		if errors.Is(err, unix.EEXIST) {
			continue
		}
		return "", nil, err
	}
	return "", nil, fmt.Errorf("unable to allocate temp file name")
}

func RenameInDir(dir *os.File, oldName, newName string, noReplace bool) error {
	if dir == nil {
		return fmt.Errorf("missing directory")
	}
	if noReplace {
		return unix.Renameat2(int(dir.Fd()), oldName, int(dir.Fd()), newName, unix.RENAME_NOREPLACE)
	}
	return unix.Renameat(int(dir.Fd()), oldName, int(dir.Fd()), newName)
}

func UnlinkInDir(dir *os.File, name string) error {
	if dir == nil {
		return fmt.Errorf("missing directory")
	}
	return unix.Unlinkat(int(dir.Fd()), name, 0)
}

func normalizeChain(rel string) ([]string, error) {
	if rel == "" || rel == "." {
		return nil, nil
	}
	clean, err := ValidateRelative(rel)
	if err != nil {
		return nil, err
	}
	if clean == "" || clean == "." {
		return nil, nil
	}
	return strings.Split(clean, "/"), nil
}

func openNoFollowDir(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func openChildDir(parent *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), filepath.Join(parent.Name(), name)), nil
}

func randomSuffix() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
