package hooks

import "io"

func ioReadAllLimit(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		limit = 64 << 10
	}
	return io.ReadAll(io.LimitReader(r, limit))
}
