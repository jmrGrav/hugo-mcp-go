package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout string, stderr string, err error)
}

type ExecRunner struct {
	MaxOutputBytes int
}

func (r ExecRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	max := r.MaxOutputBytes
	if max <= 0 {
		max = 64 * 1024
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr limitedBuffer
	stdout.limit = max
	stderr.limit = max
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return observability.RedactString(stdout.String()), observability.RedactString(stderr.String()), err
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = b.buf.Write(p[:remaining])
			b.truncated = true
		} else {
			_, _ = b.buf.Write(p)
		}
	} else {
		b.truncated = true
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	if !b.truncated {
		return b.buf.String()
	}
	return fmt.Sprintf("%s...<truncated>", b.buf.String())
}
