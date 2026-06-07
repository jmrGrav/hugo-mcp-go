package shim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
)

type fakeChild struct {
	gen          uint64
	bootstraps   int
	bootstrapErr error
	sendFunc     func(context.Context, *RPCRequest) ([]byte, error)
	closed       int
}

func (f *fakeChild) Generation() uint64 { return f.gen }

func (f *fakeChild) Bootstrap(context.Context) error {
	f.bootstraps++
	return f.bootstrapErr
}

func (f *fakeChild) Send(ctx context.Context, req *RPCRequest) ([]byte, error) {
	if f.sendFunc != nil {
		return f.sendFunc(ctx, req)
	}
	return nil, nil
}

func (f *fakeChild) Close() error {
	f.closed++
	return nil
}

func testServer(t *testing.T, child childBridge, maxBytes int64) (*Server, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	goBin := root + "/hugo-mcp-go"
	if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write go bin stub: %v", err)
	}
	workDir := root + "/work"
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	cfg := Config{
		BindAddr:         "192.168.122.69",
		BindPort:         18180,
		BackendToken:     "REDACTED",
		GoBin:            goBin,
		GoWorkDir:        workDir,
		RequestTimeoutMS: 100,
		StartupTimeoutMS: 100,
		MaxRequestBytes:  maxBytes,
		LogLevel:         "info",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("cfg.Validate() error = %v", err)
	}
	logBuf := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	srv, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	srv.child = child
	return srv, logBuf
}

func TestConfigValidation(t *testing.T) {
	t.Run("missing bind addr", func(t *testing.T) {
		cfg := Config{BindPort: 18180, BackendToken: "x", GoBin: "/x", GoWorkDir: "/x", RequestTimeoutMS: 1, StartupTimeoutMS: 1, MaxRequestBytes: 1, LogLevel: "info"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing port", func(t *testing.T) {
		cfg := Config{BindAddr: "127.0.0.1", BackendToken: "x", GoBin: "/x", GoWorkDir: "/x", RequestTimeoutMS: 1, StartupTimeoutMS: 1, MaxRequestBytes: 1, LogLevel: "info"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing token", func(t *testing.T) {
		cfg := Config{BindAddr: "127.0.0.1", BindPort: 18180, GoBin: "/x", GoWorkDir: "/x", RequestTimeoutMS: 1, StartupTimeoutMS: 1, MaxRequestBytes: 1, LogLevel: "info"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing bin", func(t *testing.T) {
		cfg := Config{BindAddr: "127.0.0.1", BindPort: 18180, BackendToken: "x", GoWorkDir: "/x", RequestTimeoutMS: 1, StartupTimeoutMS: 1, MaxRequestBytes: 1, LogLevel: "info"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing workdir", func(t *testing.T) {
		cfg := Config{BindAddr: "127.0.0.1", BindPort: 18180, BackendToken: "x", GoBin: "/x", RequestTimeoutMS: 1, StartupTimeoutMS: 1, MaxRequestBytes: 1, LogLevel: "info"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("absent bin path", func(t *testing.T) {
		root := t.TempDir()
		work := t.TempDir()
		cfg := Config{BindAddr: "127.0.0.1", BindPort: 18180, BackendToken: "x", GoBin: root + "/missing", GoWorkDir: work, RequestTimeoutMS: 1, StartupTimeoutMS: 1, MaxRequestBytes: 1, LogLevel: "info"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("absent workdir path", func(t *testing.T) {
		root := t.TempDir()
		cfg := Config{BindAddr: "127.0.0.1", BindPort: 18180, BackendToken: "x", GoBin: root + "/bin", GoWorkDir: root + "/missing", RequestTimeoutMS: 1, StartupTimeoutMS: 1, MaxRequestBytes: 1, LogLevel: "info"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestHTTPWrongPathAndMethod(t *testing.T) {
	srv, _ := testServer(t, &fakeChild{}, 1024)

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestHTTPAuthAndSizeLimits(t *testing.T) {
	srv, _ := testServer(t, &fakeChild{}, 16)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping","params":{"x":"0123456789"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rr.Code)
	}
}

func TestHTTPMalformedAndNotification(t *testing.T) {
	child := &fakeChild{
		sendFunc: func(ctx context.Context, req *RPCRequest) ([]byte, error) {
			return nil, nil
		},
	}
	srv, _ := testServer(t, child, 1024)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rr.Code)
	}
}

func TestMCPProtocolParityInitializeAndDiscovery(t *testing.T) {
	child := &fakeChild{
		sendFunc: func(ctx context.Context, req *RPCRequest) ([]byte, error) {
			id := string(req.ID)
			if strings.TrimSpace(id) == "" {
				id = `null`
			}
			if req.Method == "does/not_exist" {
				return []byte(`{"jsonrpc":"2.0","id":` + id + `,"error":{"code":-32601,"message":"Method not found: does/not_exist"}}`), nil
			}
			return []byte(`{"jsonrpc":"2.0","id":` + id + `,"result":{"ok":true}}`), nil
		},
	}
	srv, _ := testServer(t, child, 1024)

	cases := []struct {
		name     string
		body     string
		wantCode int
		wantSub  string
	}{
		{name: "initialize", body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{"roots":{"listChanged":true}},"clientInfo":{"name":"audit-client","version":"0.1.0"}}}`, wantCode: http.StatusOK, wantSub: `"protocolVersion":"2025-03-26"`},
		{name: "tools pre-init", body: `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, wantCode: http.StatusOK, wantSub: `"id":2`},
		{name: "tools invalid params", body: `{"jsonrpc":"2.0","id":3,"method":"tools/list","params":123}`, wantCode: http.StatusOK, wantSub: `"id":3`},
		{name: "resources list", body: `{"jsonrpc":"2.0","id":4,"method":"resources/list","params":{}}`, wantCode: http.StatusOK, wantSub: `"code":-32601`},
		{name: "prompts list", body: `{"jsonrpc":"2.0","id":5,"method":"prompts/list","params":{}}`, wantCode: http.StatusOK, wantSub: `"code":-32601`},
		{name: "unknown method", body: `{"jsonrpc":"2.0","id":6,"method":"does/not_exist","params":{}}`, wantCode: http.StatusOK, wantSub: `"code":-32601`},
		{name: "notification no id", body: `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`, wantCode: http.StatusAccepted, wantSub: ""},
		{name: "notification with id", body: `{"jsonrpc":"2.0","id":7,"method":"notifications/initialized","params":{}}`, wantCode: http.StatusOK, wantSub: `"code":-32600`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer REDACTED")
			rr := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rr, req)
			if rr.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantCode)
			}
			if tc.wantSub != "" && !strings.Contains(rr.Body.String(), tc.wantSub) {
				t.Fatalf("response = %s, want substring %q", rr.Body.String(), tc.wantSub)
			}
		})
	}
	if child.bootstraps == 0 {
		t.Fatal("expected child bootstrap to be called")
	}
}

func TestHTTPRequestIdPreserved(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "numeric id", body: `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`, want: `"id":1`},
		{name: "string id", body: `{"jsonrpc":"2.0","id":"abc","method":"tools/list","params":{}}`, want: `"id":"abc"`},
		{name: "null id", body: `{"jsonrpc":"2.0","id":null,"method":"tools/list","params":{}}`, want: `"id":null`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			child := &fakeChild{
				sendFunc: func(ctx context.Context, req *RPCRequest) ([]byte, error) {
					return []byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":{"ok":true}}`), nil
				},
			}
			srv, _ := testServer(t, child, 1024)

			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer REDACTED")
			rr := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			if !strings.Contains(rr.Body.String(), tc.want) {
				t.Fatalf("response = %s, want id preserved %s", rr.Body.String(), tc.want)
			}
		})
	}
}

func TestHTTPChildUnavailableAndTimeout(t *testing.T) {
	child := &fakeChild{
		sendFunc: func(ctx context.Context, req *RPCRequest) ([]byte, error) {
			return nil, context.DeadlineExceeded
		},
	}
	srv, _ := testServer(t, child, 1024)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504", rr.Code)
	}

	child.sendFunc = func(ctx context.Context, req *RPCRequest) ([]byte, error) {
		return nil, io.EOF
	}
	req = httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway && rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 502/503", rr.Code)
	}
}

func TestHTTPBootstrapFailureAndInvalidChildResponse(t *testing.T) {
	child := &fakeChild{
		bootstrapErr: errors.New("backend unavailable: down"),
	}
	srv, _ := testServer(t, child, 1024)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}

	child.bootstrapErr = nil
	child.sendFunc = func(ctx context.Context, req *RPCRequest) ([]byte, error) {
		return []byte("not-json"), nil
	}
	req = httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rr.Code)
	}
}

func TestHTTPChildRestartAndLogs(t *testing.T) {
	callCount := 0
	child := &fakeChild{
		sendFunc: func(ctx context.Context, req *RPCRequest) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return nil, context.Canceled
			}
			return []byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`), nil
		},
	}
	srv, logBuf := testServer(t, child, 1024)

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"token":"secret","path":"/home/jm/hugo-site"}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway && rr.Code != http.StatusServiceUnavailable && rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want child error", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer REDACTED")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	logs := logBuf.String()
	for _, forbidden := range []string{"secret", "/home/jm/"} {
		if strings.Contains(logs, forbidden) {
			t.Fatalf("logs leaked %q: %s", forbidden, logs)
		}
	}
	if !strings.Contains(logs, "shim request") {
		t.Fatalf("logs missing request entry: %s", logs)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	root := t.TempDir()
	work := t.TempDir()
	bin := root + "/hugo-mcp-go"
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write bin: %v", err)
	}
	t.Setenv("HUGO_MCP_SHIM_BIND_ADDR", "192.168.122.69")
	t.Setenv("HUGO_MCP_SHIM_BIND_PORT", "18180")
	t.Setenv("HUGO_MCP_SHIM_BACKEND_TOKEN", "REDACTED")
	t.Setenv("HUGO_MCP_GO_BIN", bin)
	t.Setenv("HUGO_MCP_GO_WORKDIR", work)
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.BindAddr != "192.168.122.69" || cfg.BindPort != 18180 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestServerHelpersAndConstruction(t *testing.T) {
	srv, _ := testServer(t, &fakeChild{}, 1024)

	if !srv.authorize("Bearer REDACTED") {
		t.Fatal("authorize() rejected valid token")
	}
	if srv.authorize("Bearer wrong") {
		t.Fatal("authorize() accepted invalid token")
	}

	srv.queue = make(chan struct{}, 1)
	if !srv.acquire() {
		t.Fatal("acquire() failed")
	}
	if srv.acquire() {
		t.Fatal("acquire() unexpectedly allowed overflow")
	}
	srv.release()

	if got := translateChildStatus(context.DeadlineExceeded); got != http.StatusGatewayTimeout {
		t.Fatalf("translateChildStatus(deadline) = %d", got)
	}
	if got := translateChildStatus(io.EOF); got != http.StatusBadGateway {
		t.Fatalf("translateChildStatus(eof) = %d", got)
	}
	if got := translateChildStatus(nil); got != http.StatusOK {
		t.Fatalf("translateChildStatus(nil) = %d", got)
	}

	initResp := fixedInitializeResponse(nil)
	if initResp["id"] != nil {
		t.Fatalf("fixedInitializeResponse(nil) id = %#v", initResp["id"])
	}
	encoded, err := json.Marshal(initResp)
	if err != nil {
		t.Fatalf("marshal init response: %v", err)
	}
	if !strings.Contains(string(encoded), `"protocolVersion":"2025-03-26"`) {
		t.Fatalf("fixedInitializeResponse missing protocolVersion: %s", string(encoded))
	}

	rr := httptest.NewRecorder()
	if n := writeJSON(rr, map[string]any{"ok": true}); n == 0 {
		t.Fatal("writeJSON() wrote zero bytes")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("writeJSON() status = %d", rr.Code)
	}

	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{name: "backend unavailable", err: errors.New("backend unavailable: down"), want: http.StatusServiceUnavailable},
		{name: "child stopped", err: errors.New("child stopped"), want: http.StatusServiceUnavailable},
		{name: "overloaded", err: errors.New("overloaded"), want: http.StatusTooManyRequests},
		{name: "write child", err: errors.New("write child request: boom"), want: http.StatusBadGateway},
		{name: "default", err: errors.New("unexpected"), want: http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := translateChildStatus(tc.err); got != tc.want {
				t.Fatalf("translateChildStatus(%v) = %d want %d", tc.err, got, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{name: "notification", raw: nil, want: "notification"},
		{name: "null", raw: json.RawMessage(`null`), want: "null"},
		{name: "string", raw: json.RawMessage(`"abc"`), want: "string"},
		{name: "number", raw: json.RawMessage(`1`), want: "number"},
		{name: "other", raw: json.RawMessage(`true`), want: "other"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectIDType(tc.raw); got != tc.want {
				t.Fatalf("detectIDType(%s) = %q want %q", string(tc.raw), got, tc.want)
			}
		})
	}

	errRR := httptest.NewRecorder()
	if n := writeJSON(errRR, map[string]any{"bad": make(chan int)}); n != 0 {
		t.Fatalf("writeJSON(error) wrote %d bytes want 0", n)
	}
	if errRR.Code != http.StatusInternalServerError {
		t.Fatalf("writeJSON(error) status = %d want 500", errRR.Code)
	}
}

func TestNewServerNilLoggerAndClose(t *testing.T) {
	root := t.TempDir()
	goBin := root + "/hugo-mcp-go"
	if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write go bin stub: %v", err)
	}
	workDir := root + "/work"
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	cfg := Config{
		BindAddr:         "127.0.0.1",
		BindPort:         18180,
		BackendToken:     "REDACTED",
		GoBin:            goBin,
		GoWorkDir:        workDir,
		RequestTimeoutMS: 100,
		StartupTimeoutMS: 100,
		MaxRequestBytes:  1024,
		LogLevel:         "info",
	}
	svc, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewServer() returned nil server")
	}
	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNewServerRejectsInvalidConfig(t *testing.T) {
	srv, err := NewServer(Config{}, nil)
	if err == nil {
		t.Fatal("NewServer() expected config error")
	}
	if srv != nil {
		t.Fatalf("NewServer() returned unexpected server: %#v", srv)
	}
}

func TestServerCloseWithHTTPServer(t *testing.T) {
	root := t.TempDir()
	goBin := root + "/hugo-mcp-go"
	if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write go bin stub: %v", err)
	}
	workDir := root + "/work"
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	cfg := Config{
		BindAddr:         "127.0.0.1",
		BindPort:         18180,
		BackendToken:     "REDACTED",
		GoBin:            goBin,
		GoWorkDir:        workDir,
		RequestTimeoutMS: 100,
		StartupTimeoutMS: 100,
		MaxRequestBytes:  1024,
		LogLevel:         "info",
	}
	srv, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	srv.server = &http.Server{Addr: "127.0.0.1:0"}
	if err := srv.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestListenAndServeShutdown(t *testing.T) {
	srv := &Server{
		cfg: Config{
			BindAddr: "127.0.0.1",
			BindPort: 0,
		},
		child: &fakeChild{},
		log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()
	time.Sleep(50 * time.Millisecond)
	if err := srv.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "Server closed") && !strings.Contains(err.Error(), "closed network connection") {
			t.Fatalf("ListenAndServe() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListenAndServe() did not exit")
	}
}

func TestLogRedactionHelper(t *testing.T) {
	got := observability.RedactString("Bearer secret /home/jm/hugo-site token")
	for _, forbidden := range []string{"secret", "/home/jm/"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("redaction leaked %q in %q", forbidden, got)
		}
	}
}
