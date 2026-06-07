package hooks

import (
	"crypto/rand"
	"encoding/hex"
	"os"
)

var randRead = rand.Read

func osMkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func newID() string {
	var raw [16]byte
	if _, err := randRead(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return hex.EncodeToString([]byte("hooks-fallback-id"))
}
