package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearerRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (b bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = clone.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(clone)
}

func TestHTTPNativeStreamingRouteDisabledByDefault(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, nil)
	h := newHTTPHandler(&Service{server: server}, transport.Config{
		Transport:        "http",
		HTTPToken:        "local-token",
		MaxRequestBytes:  512,
		MaxChunkBytes:    256 << 10,
		MaxResponseBytes: 1 << 20,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rr := nativeHTTPRequest(t, h, http.MethodGet, "/mcp/events", "", map[string]string{
		"Authorization": "Bearer local-token",
	})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("streaming disabled status = %d want 404", rr.Code)
	}
}

func TestHTTPNativeStreamingProgressAndWarning(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, nil)
	type buildInput struct {
		PurgeCF bool `json:"purge_cf,omitempty"`
	}
	type buildOutput struct {
		Status string `json:"status"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "build_site", Description: "Build the site."}, func(ctx context.Context, req *mcp.CallToolRequest, _ buildInput) (*mcp.CallToolResult, buildOutput, error) {
		sendProgress(ctx, req, "build_site started", 0, 3)
		sendProgress(ctx, req, "build_site completed", 3, 3)
		return nil, buildOutput{Status: "built"}, nil
	})
	mcp.AddTool(server, &mcp.Tool{Name: "check_sri_versions", Description: "Audit SRI."}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct {
		AutoFix *bool `json:"auto_fix,omitempty"`
		DryRun  *bool `json:"dry_run,omitempty"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		sendProgress(ctx, req, "check_sri_versions started", 0, 2)
		sendProgress(ctx, req, "check_sri_versions warning: SRI audit disabled", 1, 2)
		sendProgress(ctx, req, "check_sri_versions completed", 2, 2)
		return nil, map[string]any{"status": "ok"}, nil
	})

	h := newHTTPHandler(&Service{server: server}, transport.Config{
		Transport:        "http",
		StreamingEnabled: true,
		HTTPToken:        "local-token",
		MaxRequestBytes:  512,
		MaxChunkBytes:    256 << 10,
		MaxResponseBytes: 1 << 20,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(h)
	defer ts.Close()

	var events []string
	var mu sync.Mutex
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "0.1"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			mu.Lock()
			events = append(events, req.Params.Message)
			mu.Unlock()
		},
	})
	transportClient := &mcp.StreamableClientTransport{
		Endpoint: ts.URL + "/mcp/events",
		HTTPClient: &http.Client{
			Transport: bearerRoundTripper{token: "local-token", base: http.DefaultTransport},
		},
	}
	session, err := client.Connect(context.Background(), transportClient, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()

	buildParams := &mcp.CallToolParams{Name: "build_site", Arguments: map[string]any{"purge_cf": false}}
	buildParams.SetProgressToken("progress-1")
	buildRes, err := session.CallTool(context.Background(), buildParams)
	if err != nil {
		t.Fatalf("CallTool(build_site) error = %v", err)
	}
	if buildRes.StructuredContent == nil {
		t.Fatal("build_site missing structuredContent")
	}
	raw, err := json.Marshal(buildRes.StructuredContent)
	if err != nil {
		t.Fatalf("marshal build_site structured content: %v", err)
	}
	var buildGot map[string]any
	if err := json.Unmarshal(raw, &buildGot); err != nil {
		t.Fatalf("decode build_site structured content: %v", err)
	}
	if buildGot["status"] != "built" {
		t.Fatalf("build_site status = %v want built", buildGot["status"])
	}

	sriParams := &mcp.CallToolParams{Name: "check_sri_versions", Arguments: map[string]any{"auto_fix": false, "dry_run": true}}
	sriParams.SetProgressToken("progress-2")
	sriRes, err := session.CallTool(context.Background(), sriParams)
	if err != nil {
		t.Fatalf("CallTool(check_sri_versions) error = %v", err)
	}
	if sriRes.IsError {
		t.Fatalf("check_sri_versions returned error: %#v", sriRes)
	}

	if eventCount(&mu, &events) == 0 {
		t.Fatal("expected progress notifications")
	}
	if !waitForEvent(t, &mu, &events, "build_site started", 500*time.Millisecond) {
		t.Fatalf("missing build_site started event: %#v", snapshotEvents(&mu, &events))
	}
	if !waitForEvent(t, &mu, &events, "build_site completed", 500*time.Millisecond) {
		t.Fatalf("missing build_site completed event: %#v", snapshotEvents(&mu, &events))
	}
	if !waitForEvent(t, &mu, &events, "check_sri_versions warning: SRI audit disabled", 500*time.Millisecond) {
		t.Fatalf("missing warning event: %#v", snapshotEvents(&mu, &events))
	}
}

func waitForEvent(t *testing.T, mu *sync.Mutex, events *[]string, want string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		mu.Lock()
		found := false
		for _, v := range *events {
			if strings.Contains(v, want) {
				found = true
				break
			}
		}
		mu.Unlock()
		if found {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func eventCount(mu *sync.Mutex, events *[]string) int {
	mu.Lock()
	defer mu.Unlock()
	return len(*events)
}

func snapshotEvents(mu *sync.Mutex, events *[]string) []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, len(*events))
	copy(out, *events)
	return out
}

func sendProgress(ctx context.Context, req *mcp.CallToolRequest, message string, progress, total float64) {
	if req == nil || req.Session == nil || req.Params.GetProgressToken() == nil {
		return
	}
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		Message:       message,
		ProgressToken: req.Params.GetProgressToken(),
		Progress:      progress,
		Total:         total,
	})
}
