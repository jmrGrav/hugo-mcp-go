package shim

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func childEnv(cfg Config) []string {
	path := "/usr/local/bin:/usr/bin:/bin"
	if strings.TrimSpace(cfg.ChildPath) != "" {
		path = strings.TrimSpace(cfg.ChildPath) + ":" + path
	}
	return []string{
		"HUGO_ROOT=/var/lib/hugo-mcp-go",
		"HUGO_CONTENT_ROOT=/var/lib/hugo-mcp-go/content",
		"HUGO_STATIC_ROOT=/var/lib/hugo-mcp-go/static",
		"HUGO_MAX_REQUEST_BYTES=1048576",
		"HUGO_MAX_TOOL_ARGS_BYTES=262144",
		"HUGO_MAX_PAGE_BYTES=1048576",
		"HUGO_MAX_ASSET_BYTES=26214400",
		"HUGO_MAX_LIST_PAGES=500",
		"HUGO_MAX_LIST_ASSETS=500",
		"PATH=" + path,
	}
}

type rawChild struct {
	cfg Config

	mu           sync.Mutex
	cond         *sync.Cond
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	pending      map[string]chan []byte
	writeMu      sync.Mutex
	starting     bool
	alive        bool
	bootstrapped bool
	reason       string
	stop         bool
	generation   uint64
	seq          uint64
}

func (c *rawChild) Generation() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

func newRawChild(cfg Config) *rawChild {
	c := &rawChild{
		cfg:     cfg,
		pending: make(map[string]chan []byte),
	}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *rawChild) ensure(ctx context.Context) error {
	c.mu.Lock()
	if c.alive {
		c.mu.Unlock()
		return nil
	}
	if c.starting {
		for c.starting && !c.alive && !c.stop {
			c.cond.Wait()
		}
		defer c.mu.Unlock()
		if c.alive {
			return nil
		}
		if c.stop {
			return fmt.Errorf("child stopped")
		}
		return fmt.Errorf("backend unavailable: %s", c.reason)
	}
	c.starting = true
	c.mu.Unlock()

	err := c.spawnWithBackoff(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.starting = false
	if err != nil {
		c.reason = err.Error()
		c.cond.Broadcast()
		return err
	}
	c.alive = true
	c.reason = ""
	c.generation++
	c.cond.Broadcast()
	return nil
}

func (c *rawChild) Bootstrap(ctx context.Context) error {
	if err := c.ensure(ctx); err != nil {
		return err
	}

	c.mu.Lock()
	if c.bootstrapped {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	initReq := &RPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"clientInfo":{"name":"hugo-mcp","version":"1.0.0"}}`),
	}
	if _, err := c.sendRaw(ctx, initReq); err != nil {
		return err
	}
	notifyReq := &RPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		Params:  json.RawMessage(`{}`),
	}
	if _, err := c.sendRaw(ctx, notifyReq); err != nil {
		return err
	}

	c.mu.Lock()
	c.bootstrapped = true
	c.mu.Unlock()
	return nil
}

func (c *rawChild) spawnWithBackoff(ctx context.Context) error {
	delay := time.Duration(DefaultRestartBaseDelayMS) * time.Millisecond
	var lastErr error
	for attempt := 0; attempt < DefaultRestartMaxRetries; attempt++ {
		lastErr = c.spawn(ctx)
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > time.Duration(DefaultRestartMaxDelayMS)*time.Millisecond {
			delay = time.Duration(DefaultRestartMaxDelayMS) * time.Millisecond
		}
	}
	return lastErr
}

func (c *rawChild) spawn(ctx context.Context) error {
	cmdCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(cmdCtx, c.cfg.GoBin)
	cmd.Dir = c.cfg.GoWorkDir
	cmd.Env = childEnv(c.cfg)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		cancel()
		_ = stdin.Close()
		_ = stdout.Close()
		return fmt.Errorf("start child: %w", err)
	}

	c.mu.Lock()
	c.cmd = cmd
	c.cancel = cancel
	c.stdin = stdin
	c.stdout = stdout
	c.mu.Unlock()

	go c.readLoop(cmd, stdout)
	go c.waitLoop(cmd)

	if err := ctx.Err(); err != nil {
		cancel()
		return err
	}
	return nil
}

func (c *rawChild) sendRaw(ctx context.Context, req *RPCRequest) ([]byte, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')

	expectResponse := len(req.ID) > 0
	var respCh chan []byte
	key := idKey(req.ID)
	if expectResponse {
		respCh = make(chan []byte, 1)
		c.mu.Lock()
		if !c.alive {
			c.mu.Unlock()
			return nil, fmt.Errorf("backend unavailable: %s", c.reason)
		}
		c.pending[key] = respCh
		stdin := c.stdin
		c.mu.Unlock()

		if stdin == nil {
			return nil, fmt.Errorf("backend unavailable: missing stdin")
		}
		c.writeMu.Lock()
		_, werr := stdin.Write(raw)
		c.writeMu.Unlock()
		if werr != nil {
			c.mu.Lock()
			delete(c.pending, key)
			c.mu.Unlock()
			return nil, fmt.Errorf("write child request: %w", werr)
		}
		select {
		case resp := <-respCh:
			if resp == nil {
				return nil, fmt.Errorf("backend unavailable: %s", c.reason)
			}
			return resp, nil
		case <-ctx.Done():
			c.mu.Lock()
			delete(c.pending, key)
			c.mu.Unlock()
			return nil, ctx.Err()
		}
	}

	c.mu.Lock()
	stdin := c.stdin
	c.mu.Unlock()
	if stdin == nil {
		return nil, fmt.Errorf("backend unavailable: missing stdin")
	}
	c.writeMu.Lock()
	_, err = stdin.Write(raw)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("write child notification: %w", err)
	}
	return nil, nil
}

func (c *rawChild) waitLoop(cmd *exec.Cmd) {
	err := cmd.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != cmd {
		return
	}
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if err != nil {
		c.reason = err.Error()
	} else {
		c.reason = "child exited"
	}
	c.alive = false
	for key, ch := range c.pending {
		delete(c.pending, key)
		select {
		case ch <- nil:
		default:
		}
		close(ch)
	}
	c.cond.Broadcast()
}

func (c *rawChild) readLoop(cmd *exec.Cmd, stdout io.ReadCloser) {
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				c.failRead(cmd, err)
			}
			return
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		msg, err := jsonrpc.DecodeMessage(line)
		if err != nil {
			continue
		}
		switch m := msg.(type) {
		case *jsonrpc.Response:
			key := idKeyJSON(m.ID.Raw())
			c.mu.Lock()
			ch := c.pending[key]
			if ch != nil {
				delete(c.pending, key)
			}
			c.mu.Unlock()
			if ch != nil {
				select {
				case ch <- append([]byte(nil), line...):
				default:
				}
				close(ch)
			}
		case *jsonrpc.Request:
			// The backend should not ask the shim to act as an MCP client in this staging model.
			// Ignore unexpected server-initiated requests rather than deadlock the bridge.
		}
	}
}

func (c *rawChild) failRead(cmd *exec.Cmd, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != cmd {
		return
	}
	c.reason = err.Error()
	c.alive = false
	for key, ch := range c.pending {
		delete(c.pending, key)
		select {
		case ch <- nil:
		default:
		}
		close(ch)
	}
	c.cond.Broadcast()
}

func (c *rawChild) Close() error {
	c.mu.Lock()
	c.stop = true
	cmd := c.cmd
	cancel := c.cancel
	stdin := c.stdin
	c.mu.Unlock()
	c.cond.Broadcast()
	if cancel != nil {
		cancel()
	}
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}

func (c *rawChild) Send(ctx context.Context, req *RPCRequest) ([]byte, error) {
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}

	if err := c.Bootstrap(ctx); err != nil {
		return nil, err
	}

	next := cloneRequest(req)
	if strings.TrimSpace(next.Method) == "tools/list" {
		next.Params = json.RawMessage(`{}`)
	}

	rewriteNull := false
	if isNullID(next.ID) {
		rewriteNull = true
		next.ID = json.RawMessage(strconv.Quote(c.nextSyntheticID()))
	}

	resp, err := c.sendRaw(ctx, next)
	if err != nil {
		return nil, err
	}
	if rewriteNull {
		resp = rewriteResponseID(resp, nil)
	}
	if rewritten, ok := rewriteUnsupportedMethod(resp, next.Method); ok {
		return rewritten, nil
	}
	return resp, nil
}

func idKeyJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(raw)
}

func idKey(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "<notification>"
	}
	return string(bytes.TrimSpace(raw))
}

func (c *rawChild) nextSyntheticID() string {
	return fmt.Sprintf("shim-%d", atomic.AddUint64(&c.seq, 1))
}

func cloneRequest(req *RPCRequest) *RPCRequest {
	if req == nil {
		return nil
	}
	cp := *req
	if req.ID != nil {
		cp.ID = append(json.RawMessage(nil), req.ID...)
	}
	if req.Params != nil {
		cp.Params = append(json.RawMessage(nil), req.Params...)
	}
	return &cp
}

func isNullID(id json.RawMessage) bool {
	return strings.TrimSpace(string(id)) == "null"
}

func rewriteResponseID(resp []byte, id json.RawMessage) []byte {
	var env map[string]any
	if err := json.Unmarshal(resp, &env); err != nil {
		return resp
	}
	if len(id) == 0 {
		env["id"] = nil
	} else {
		env["id"] = json.RawMessage(append([]byte(nil), id...))
	}
	out, err := json.Marshal(env)
	if err != nil {
		return resp
	}
	return out
}

func rewriteUnsupportedMethod(resp []byte, method string) ([]byte, bool) {
	var env map[string]any
	if err := json.Unmarshal(resp, &env); err != nil {
		return nil, false
	}
	rawErr, ok := env["error"].(map[string]any)
	if !ok {
		return nil, false
	}
	code, _ := rawErr["code"].(float64)
	msg, _ := rawErr["message"].(string)
	if code != 0 {
		return nil, false
	}
	lowered := strings.ToLower(msg)
	if !strings.Contains(lowered, "unsupported") && !strings.Contains(lowered, "not handled") {
		return nil, false
	}
	env["error"] = map[string]any{
		"code":    -32601,
		"message": fmt.Sprintf("Method not found: %s", method),
	}
	out, err := json.Marshal(env)
	if err != nil {
		return nil, false
	}
	return out, true
}
