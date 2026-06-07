package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func nativeHTTPTestHandler(t *testing.T) http.Handler {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, nil)

	type buildInput struct {
		PurgeCF bool `json:"purge_cf,omitempty"`
	}
	type buildOutput struct {
		Status string `json:"status"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "build_site", Description: "Build the Hugo site."}, func(ctx context.Context, _ *mcp.CallToolRequest, _ buildInput) (*mcp.CallToolResult, buildOutput, error) {
		return nil, buildOutput{Status: "built"}, nil
	})

	svc := &Service{server: server}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := transport.Config{
		Transport:        "http",
		HTTPBindAddr:     "127.0.0.1",
		HTTPBindPort:     18181,
		HTTPToken:        "local-token",
		MaxRequestBytes:  512,
		MaxChunkBytes:    256 << 10,
		MaxResponseBytes: 1 << 20,
	}
	return newHTTPHandler(svc, cfg, logger)
}

func nativeHTTPRequest(t *testing.T, h http.Handler, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestHTTPNativeAuthAndMethodGuards(t *testing.T) {
	h := nativeHTTPTestHandler(t)

	rr := nativeHTTPRequest(t, h, http.MethodGet, "/mcp", "", nil)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /mcp status = %d want 405", rr.Code)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	rr = nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, headers)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing bearer status = %d want 401", rr.Code)
	}

	headers["Authorization"] = "Bearer wrong"
	rr = nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, headers)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong bearer status = %d want 401", rr.Code)
	}

	headers["Authorization"] = "Bearer local-token"
	headers["Content-Type"] = "text/plain"
	rr = nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, headers)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("wrong content-type status = %d want 415", rr.Code)
	}

	headers["Content-Type"] = "application/json"
	rr = nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"x":"`+strings.Repeat("a", 2048)+`"}}`, headers)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d want 413", rr.Code)
	}
}

func TestHTTPNativeInitializeAndToolsList(t *testing.T) {
	h := nativeHTTPTestHandler(t)

	headers := map[string]string{
		"Authorization": "Bearer local-token",
		"Content-Type":  "application/json",
	}
	rr := nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{"roots":{"listChanged":true}},"clientInfo":{"name":"test","version":"0.1.0"}}}`, headers)
	if rr.Code != http.StatusOK {
		t.Fatalf("initialize status = %d want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"protocolVersion":"2025-03-26"`) {
		t.Fatalf("initialize response missing protocolVersion: %s", rr.Body.String())
	}

	rr = nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`, headers)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("notification status = %d want 202", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("notification body = %q want empty", rr.Body.String())
	}

	rr = nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, headers)
	if rr.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"build_site"`) {
		t.Fatalf("tools/list response missing build_site: %s", rr.Body.String())
	}
}

func TestHTTPNativeBuildSiteStructuredContent(t *testing.T) {
	h := nativeHTTPTestHandler(t)

	headers := map[string]string{
		"Authorization": "Bearer local-token",
		"Content-Type":  "application/json",
	}
	rr := nativeHTTPRequest(t, h, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"build_site","arguments":{"purge_cf":false}}}`, headers)
	if rr.Code != http.StatusOK {
		t.Fatalf("build_site status = %d want 200", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %#v", resp)
	}
	sc, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("missing structuredContent: %#v", result)
	}
	if got := sc["status"]; got != "built" {
		t.Fatalf("structuredContent.status = %v want built", got)
	}
}

func TestHTTPNativeRedactsLogs(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, nil)
	svc := &Service{server: server}
	buf := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := transport.Config{
		Transport:        "http",
		HTTPBindAddr:     "127.0.0.1",
		HTTPBindPort:     18181,
		HTTPToken:        "local-token",
		MaxRequestBytes:  512,
		MaxChunkBytes:    256 << 10,
		MaxResponseBytes: 1 << 20,
	}
	h := newHTTPHandler(svc, cfg, logger)

	headers := map[string]string{
		"Authorization": "Bearer local-token",
		"Content-Type":  "application/json",
	}
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"test","version":"0.1.0","token":"secret"},"path":"/home/jm/private"}}`
	rr := nativeHTTPRequest(t, h, http.MethodPost, "/mcp", body, headers)
	if rr.Code != http.StatusOK {
		t.Fatalf("initialize status = %d want 200", rr.Code)
	}
	logs := buf.String()
	for _, forbidden := range []string{"secret", "/home/jm/", "local-token"} {
		if strings.Contains(logs, forbidden) {
			t.Fatalf("logs leaked %q: %s", forbidden, logs)
		}
	}
}
