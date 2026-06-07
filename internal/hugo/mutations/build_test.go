package mutations

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/runner"
)

func TestBuildSiteNominal(t *testing.T) {
	ws := newTestWorkspace(t)
	fr := &fakeRunner{}
	svc := NewBuildService(ws, fr)
	svc.Timeout = time.Second

	got, err := svc.Build(context.Background(), BuildRequest{PurgeCF: true})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	assertBuildResult(t, got, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "build_site.response.json"))
	if fr.name != "hugo" {
		t.Fatalf("Build() command = %q want %q", fr.name, "hugo")
	}
	if !containsArg(fr.args, "--source") || !containsArg(fr.args, ws.HugoRoot) || !containsArg(fr.args, "--destination") || !containsArg(fr.args, ws.PublicRoot) {
		t.Fatalf("Build() args = %#v", fr.args)
	}
}

func TestBuildSiteTimeoutPropagates(t *testing.T) {
	ws := newTestWorkspace(t)
	fr := &fakeRunner{blockUntilCancel: true}
	svc := NewBuildService(ws, fr)
	svc.Timeout = 20 * time.Millisecond

	_, err := svc.Build(context.Background(), BuildRequest{})
	if err == nil {
		t.Fatal("Build() expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("Build() error = %v", err)
	}
}

func TestBuildSiteUsesDefaultTimeoutWhenUnset(t *testing.T) {
	ws := newTestWorkspace(t)
	fr := &fakeRunner{checkDeadline: true}
	svc := NewBuildService(ws, fr)
	svc.Timeout = 0

	if _, err := svc.Build(context.Background(), BuildRequest{}); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !fr.sawDeadline {
		t.Fatal("Build() did not apply a default timeout")
	}
}

func TestBuildSiteFailureIncludesStderr(t *testing.T) {
	ws := newTestWorkspace(t)
	fr := &fakeRunner{
		stdout: "ok",
		stderr: "build failed: <redacted-path>",
		err:    errors.New("exit status 1"),
	}
	svc := NewBuildService(ws, fr)
	svc.Timeout = time.Second

	_, err := svc.Build(context.Background(), BuildRequest{})
	if err == nil {
		t.Fatal("Build() expected error")
	}
	if !strings.Contains(err.Error(), "build_site failed") || !strings.Contains(err.Error(), "<redacted-path>") {
		t.Fatalf("Build() error = %v", err)
	}
}

func TestBuildSiteRejectsMissingWiring(t *testing.T) {
	t.Run("missing stage", func(t *testing.T) {
		svc := &BuildService{Runner: &fakeRunner{}, Timeout: time.Second}
		if _, err := svc.Build(context.Background(), BuildRequest{}); err == nil {
			t.Fatal("Build() expected error")
		}
	})
	t.Run("missing runner", func(t *testing.T) {
		svc := &BuildService{Stage: newTestWorkspace(t), Timeout: time.Second}
		if _, err := svc.Build(context.Background(), BuildRequest{}); err == nil {
			t.Fatal("Build() expected error")
		}
	})
}

type fakeRunner struct {
	name             string
	args             []string
	blockUntilCancel bool
	checkDeadline    bool
	sawDeadline      bool
	stdout           string
	stderr           string
	err              error
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	if f.checkDeadline {
		_, f.sawDeadline = ctx.Deadline()
	}
	if f.blockUntilCancel {
		<-ctx.Done()
		return "", "", ctx.Err()
	}
	return f.stdout, f.stderr, f.err
}

func assertBuildResult(t *testing.T, got BuildResult, snapshotPath string) {
	t.Helper()
	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	var wantResult map[string]any
	if err := json.Unmarshal([]byte(want.Content[0].Text), &wantResult); err != nil {
		t.Fatal(err)
	}
	gotRaw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var gotResult map[string]any
	if err := json.Unmarshal(gotRaw, &gotResult); err != nil {
		t.Fatal(err)
	}
	if !equalJSON(t, gotResult, wantResult) {
		t.Fatalf("Build() mismatch:\n got=%s\nwant=%s", string(gotRaw), want.Content[0].Text)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

var _ runner.Runner = (*fakeRunner)(nil)
