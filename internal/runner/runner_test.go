package runner

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExecRunnerRunCapturesTruncatesAndRedacts(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		helperRunnerProcess(t)
		return
	}

	r := ExecRunner{MaxOutputBytes: 16}
	ctx := context.Background()
	stdout, stderr, err := r.Run(ctx, os.Args[0], "-test.run=TestHelperProcess", "helper")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(stdout, "stdout-Bearer <redacted>") {
		t.Fatalf("stdout not redacted/captured: %q", stdout)
	}
	if !strings.Contains(stdout, "...<truncated>") {
		t.Fatalf("stdout not truncated: %q", stdout)
	}
	if !strings.Contains(stderr, "...<truncated>") {
		t.Fatalf("stderr not truncated: %q", stderr)
	}
}

func TestExecRunnerRunContextCancel(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		helperRunnerProcess(t)
		return
	}

	r := ExecRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, _, err := r.Run(ctx, os.Args[0], "-test.run=TestHelperProcess", "helper-slow")
	if err == nil {
		t.Fatal("Run() expected error")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "signal: killed") {
		t.Fatalf("unexpected Run() error: %v", err)
	}
}

func helperRunnerProcess(t *testing.T) {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "-test.run=TestHelperProcess") {
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "helper" {
		_, _ = os.Stdout.WriteString("stdout-Bearer token123 token456 token789 token000 and more padding to force truncation\n")
		_, _ = os.Stderr.WriteString("stderr-abcdefghijklmnopqrstuvwxyz-0123456789-padding-to-force-truncation\n")
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "helper-slow" {
		time.Sleep(5 * time.Second)
	}
	os.Exit(0)
}

func TestHelperProcess(t *testing.T) {
	helperRunnerProcess(t)
}
