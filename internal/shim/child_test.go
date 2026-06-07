package shim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRawChildLifecycleAndHelpers(t *testing.T) {
	script := writeChildStub(t, `#!/usr/bin/env python3
import json, sys, time
for raw in sys.stdin:
    raw = raw.strip()
    if not raw:
        continue
    req = json.loads(raw)
    method = req.get("method")
    if method == "notifications/initialized":
        continue
    if method == "slow/no-response":
        time.sleep(2)
        continue
    if method == "initialize":
        resp = {"jsonrpc":"2.0","id":req.get("id"),"result":{"ok":True,"method":method}}
    elif method == "does/not_exist":
        resp = {"jsonrpc":"2.0","id":req.get("id"),"error":{"code":0,"message":"unsupported"}}
    else:
        resp = {"jsonrpc":"2.0","id":req.get("id"),"result":{"method":method}}
    sys.stdout.write(json.dumps(resp) + "\n")
    sys.stdout.flush()
`)
	cfg := Config{GoBin: script, GoWorkDir: t.TempDir()}
	c := newRawChild(cfg)

	if got := c.Generation(); got != 0 {
		t.Fatalf("Generation() = %d want 0", got)
	}
	env := childEnv(cfg)
	if !contains(env, "HUGO_ROOT=/var/lib/hugo-mcp-go") || !contains(env, "PATH=/usr/local/bin:/usr/bin:/bin") {
		t.Fatalf("childEnv() missing expected values: %#v", env)
	}
	overrideEnv := childEnv(Config{ChildPath: "/tmp/fakebin"})
	if !contains(overrideEnv, "PATH=/tmp/fakebin:/usr/local/bin:/usr/bin:/bin") {
		t.Fatalf("childEnv() missing override PATH: %#v", overrideEnv)
	}

	ctx := context.Background()
	if err := c.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if got := c.Generation(); got == 0 {
		t.Fatal("Generation() did not advance after bootstrap")
	}

	resp, err := c.Send(ctx, &RPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Send(tools/list) error = %v", err)
	}
	assertJSONContainsAny(t, resp, `"id": 1`, `"id":1`)

	resp, err = c.Send(ctx, &RPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`null`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Send(null id) error = %v", err)
	}
	assertJSONContainsAny(t, resp, `"id": null`, `"id":null`)

	resp, err = c.Send(ctx, &RPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"abc"`),
		Method:  "does/not_exist",
		Params:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Send(unknown) error = %v", err)
	}
	assertJSONContainsAny(t, resp, `"code": -32601`, `"code":-32601`)
	assertJSONContains(t, resp, `Method not found: does/not_exist`)

	if _, err := c.sendRaw(ctx, &RPCRequest{JSONRPC: "2.0", Method: "notifications/initialized", Params: json.RawMessage(`{}`)}); err != nil {
		t.Fatalf("sendRaw(notification) error = %v", err)
	}

	slowCtx, slowCancel := context.WithTimeout(ctx, 25*time.Millisecond)
	defer slowCancel()
	if _, err := c.sendRaw(slowCtx, &RPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`77`),
		Method:  "slow/no-response",
		Params:  json.RawMessage(`{}`),
	}); err == nil || (!errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline")) {
		t.Fatalf("sendRaw timeout error = %v", err)
	}

	if got := idKeyJSON("abc"); got != `"abc"` {
		t.Fatalf("idKeyJSON() = %q want %q", got, `"abc"`)
	}
	if got := idKey(json.RawMessage(` 42 `)); got != "42" {
		t.Fatalf("idKey() = %q want 42", got)
	}
	if got := c.nextSyntheticID(); !strings.HasPrefix(got, "shim-") {
		t.Fatalf("nextSyntheticID() = %q", got)
	}
	if !isNullID(json.RawMessage(` null `)) || isNullID(json.RawMessage(`1`)) {
		t.Fatal("isNullID() mismatch")
	}
	clone := cloneRequest(&RPCRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Params: json.RawMessage(`{"x":1}`), Method: "m"})
	if clone == nil || string(clone.ID) != "1" || string(clone.Params) != `{"x":1}` {
		t.Fatalf("cloneRequest() = %#v", clone)
	}
	rewritten := rewriteResponseID([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`), nil)
	assertJSONContains(t, rewritten, `"id":null`)
	rewritten = rewriteResponseID([]byte(`{"jsonrpc":"2.0","id":"abc","result":{"ok":true}}`), json.RawMessage(`"abc"`))
	assertJSONContainsAny(t, rewritten, `"id":"abc"`, `"id": "abc"`)
	if got := rewriteResponseID([]byte(`not-json`), nil); string(got) != "not-json" {
		t.Fatalf("rewriteResponseID() changed invalid JSON: %s", string(got))
	}
	unsupported, ok := rewriteUnsupportedMethod([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":0,"message":"unsupported" }}`), "tools/list")
	if !ok {
		t.Fatal("rewriteUnsupportedMethod() did not rewrite")
	}
	assertJSONContains(t, unsupported, `"code":-32601`)
	assertJSONContains(t, unsupported, `Method not found: tools/list`)
	if _, ok := rewriteUnsupportedMethod([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"other"}}`), "tools/list"); ok {
		t.Fatal("rewriteUnsupportedMethod() unexpectedly rewrote a non-unsupported error")
	}
	if _, ok := rewriteUnsupportedMethod([]byte(`not-json`), "tools/list"); ok {
		t.Fatal("rewriteUnsupportedMethod() unexpectedly rewrote invalid JSON")
	}
	if got := idKeyJSON(make(chan int)); got == "" {
		t.Fatal("idKeyJSON() empty fallback")
	}
	if cloneRequest(nil) != nil {
		t.Fatal("cloneRequest(nil) = non-nil")
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.spawnWithBackoff(cancelCtx); err == nil {
		t.Fatal("spawnWithBackoff() expected context error")
	}

	starting := newRawChild(cfg)
	starting.mu.Lock()
	starting.starting = true
	starting.stop = true
	starting.mu.Unlock()
	if err := starting.ensure(context.Background()); err == nil || !strings.Contains(err.Error(), "child stopped") {
		t.Fatalf("ensure(starting) error = %v", err)
	}

	cmd := &exec.Cmd{}
	c.mu.Lock()
	c.cmd = cmd
	c.pending["boom"] = make(chan []byte, 1)
	c.mu.Unlock()
	c.failRead(cmd, errors.New("boom"))

	readChild := newRawChild(cfg)
	readChild.mu.Lock()
	readChild.alive = true
	readChild.pending["1"] = make(chan []byte, 1)
	ch := readChild.pending["1"]
	readChild.mu.Unlock()
	pr, pw := io.Pipe()
	go readChild.readLoop(cmd, pr)
	_, _ = io.WriteString(pw, "not-json\n")
	_, _ = io.WriteString(pw, `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`+"\n")
	_ = pw.Close()
	select {
	case got := <-ch:
		assertJSONContains(t, got, `"ok":true`)
	case <-time.After(2 * time.Second):
		t.Fatal("readLoop did not deliver response")
	}

	if got := c.Close(); got != nil {
		t.Fatalf("Close() error = %v", got)
	}
}

func TestRawChildWaitLoopAndSpawnFailure(t *testing.T) {
	script := writeChildStub(t, `#!/usr/bin/env python3
import sys, time
for raw in sys.stdin:
    raw = raw.strip()
    if not raw:
        continue
    time.sleep(0.01)
`)
	cfg := Config{GoBin: script, GoWorkDir: t.TempDir()}
	c := newRawChild(cfg)

	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start stub cmd: %v", err)
	}
	ch := make(chan []byte, 1)
	c.mu.Lock()
	c.alive = true
	c.cmd = cmd
	c.pending["1"] = ch
	c.mu.Unlock()
	c.waitLoop(cmd)
	select {
	case got := <-ch:
		if got != nil {
			t.Fatalf("waitLoop pending = %q want nil", string(got))
		}
	default:
		t.Fatal("waitLoop did not notify pending request")
	}

	bad := Config{GoBin: filepath.Join(t.TempDir(), "missing"), GoWorkDir: t.TempDir()}
	c2 := newRawChild(bad)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := c2.ensure(ctx); err == nil {
		t.Fatal("ensure() expected error for missing child binary")
	}
}

func TestRawChildRetrySpawnWithBackoff(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real-child.py")
	link := filepath.Join(dir, "child-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create broken symlink: %v", err)
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(target, []byte(`#!/usr/bin/env python3
import json, sys
for raw in sys.stdin:
    raw = raw.strip()
    if not raw:
        continue
    req = json.loads(raw)
    method = req.get("method")
    if method == "notifications/initialized":
        continue
    if method == "initialize":
        resp = {"jsonrpc":"2.0","id":req.get("id"),"result":{"ok":True}}
    else:
        resp = {"jsonrpc":"2.0","id":req.get("id"),"result":{"method":method}}
    sys.stdout.write(json.dumps(resp) + "\n")
    sys.stdout.flush()
`), 0o755)
	}()
	cfg := Config{GoBin: link, GoWorkDir: t.TempDir()}
	c := newRawChild(cfg)
	if err := c.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
}

func writeChildStub(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "child-stub.py")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

func assertJSONContains(t *testing.T, raw []byte, needle string) {
	t.Helper()
	if !bytes.Contains(raw, []byte(needle)) {
		t.Fatalf("response %s missing %q", string(raw), needle)
	}
}

func assertJSONContainsAny(t *testing.T, raw []byte, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if bytes.Contains(raw, []byte(needle)) {
			return
		}
	}
	t.Fatalf("response %s missing any of %q", string(raw), needles)
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
