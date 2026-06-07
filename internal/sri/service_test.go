package sri

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type fakeHookProcessor struct {
	events  []hooks.HookEvent
	summary hooks.HookSummary
}

func (f *fakeHookProcessor) Process(ctx context.Context, event hooks.HookEvent) hooks.HookSummary {
	f.events = append(f.events, event)
	return f.summary
}

type fakeBuilder struct {
	calls  []mutations.BuildRequest
	result mutations.BuildResult
	err    error
	build  func() error
}

func (f *fakeBuilder) Build(ctx context.Context, req mutations.BuildRequest) (mutations.BuildResult, error) {
	f.calls = append(f.calls, req)
	if f.build != nil {
		if err := f.build(); err != nil {
			return mutations.BuildResult{}, err
		}
	}
	if f.err != nil {
		return mutations.BuildResult{}, f.err
	}
	return f.result, nil
}

func TestServiceCheckReportsOKAndJSONCompatibility(t *testing.T) {
	root := writeSriFixture(t, sriFixture{
		PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`,
		LayoutContent:   `{{/* https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css */}}`,
		SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css",
		SRIHashBody:     []byte("animate-ok"),
		JSDelivrVersion: "4.1.1",
	})

	svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "data.jsdelivr.com":
			return jsonResponse(`{"tags":{"latest":"4.1.1"}}`), nil
		case req.URL.String() == "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css":
			return bodyResponse("animate-ok"), nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	})}, nil, nil)

	got, err := svc.Check(context.Background(), Request{AutoFix: boolPtr(false), DryRun: boolPtr(true)})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if got.Plugin != "sri-check" {
		t.Fatalf("Plugin = %q want sri-check", got.Plugin)
	}
	if got.Success != true {
		t.Fatalf("Success = %v want true", got.Success)
	}
	if got.ExitCode != 0 {
		t.Fatalf("ExitCode = %d want 0", got.ExitCode)
	}
	if got.AutoFixRequested {
		t.Fatal("AutoFixRequested unexpectedly true")
	}
	if !got.DryRun {
		t.Fatal("DryRun unexpectedly false")
	}
	if got.Report.Summary != "OK" {
		t.Fatalf("Report.Summary = %q want OK", got.Report.Summary)
	}
	if got.Report.Diagnostic.MinorOutdated != "0" {
		t.Fatalf("Report.Diagnostic.MinorOutdated = %q want 0", got.Report.Diagnostic.MinorOutdated)
	}
	if len(got.Report.Diagnostic.HashMismatch) != 0 {
		t.Fatalf("Report.Diagnostic.HashMismatch = %#v want empty", got.Report.Diagnostic.HashMismatch)
	}
	if got.Downstream != nil {
		t.Fatalf("Downstream = %#v want nil", got.Downstream)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(raw), `"plugin":"sri-check"`) {
		t.Fatalf("JSON output missing plugin field: %s", raw)
	}
	if !strings.Contains(string(raw), `"dry_run":true`) {
		t.Fatalf("JSON output missing dry_run field: %s", raw)
	}
}

func TestServiceCheckAutoFixAppliesAndTriggersHooks(t *testing.T) {
	root := writeSriFixture(t, sriFixture{
		PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`,
		LayoutContent:   `{{/* https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css */}}`,
		SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css",
		SRIHashBody:     []byte("animate-old"),
		JSDelivrVersion: "4.1.1",
	})
	hook := &fakeHookProcessor{summary: hooks.HookSummary{
		HooksEnabled:    true,
		QueuedURLsCount: 1,
		CloudflarePurge: hooks.HookRunResult{Provider: "cloudflare", Status: "ok", URLCount: 1, Attempts: 1},
		GoogleIndexing:  hooks.HookRunResult{Provider: "google_indexing", Status: "dry_run", URLCount: 1, DryRun: true},
		IndexNow:        hooks.HookRunResult{Provider: "indexnow", Status: "dry_run", URLCount: 1, DryRun: true},
	}}
	builder := &fakeBuilder{
		result: mutations.BuildResult{Status: "built", Deploy: "DEPLOY_SKIPPED"},
		build: func() error {
			for _, path := range []string{
				filepath.Join(root, "public", "index.html"),
				filepath.Join(root, "themes", "LoveIt", "layouts", "_partials", "head.html"),
			} {
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				raw = bytes.ReplaceAll(raw, []byte("4.1.1"), []byte("4.1.2"))
				if err := os.WriteFile(path, raw, 0o644); err != nil {
					return err
				}
			}
			return nil
		},
	}
	svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "data.jsdelivr.com":
			return jsonResponse(`{"tags":{"latest":"4.1.2"}}`), nil
		case req.URL.String() == "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css":
			return bodyResponse("animate-old"), nil
		case req.URL.String() == "https://cdn.jsdelivr.net/npm/animate.css@4.1.2/animate.min.css":
			return bodyResponse("animate-new"), nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	})}, hook, builder)

	got, err := svc.Check(context.Background(), Request{AutoFix: boolPtr(true), DryRun: boolPtr(false)})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if got.ExitCode != 0 {
		t.Fatalf("ExitCode = %d want 0", got.ExitCode)
	}
	if got.Report.Summary != "OK" {
		t.Fatalf("Report.Summary = %q want OK", got.Report.Summary)
	}
	if len(got.Report.AutoFix.Applied) != 1 {
		t.Fatalf("Applied = %#v want one entry", got.Report.AutoFix.Applied)
	}
	if len(builder.calls) != 1 {
		t.Fatalf("Build() calls = %d want 1", len(builder.calls))
	}
	if len(hook.events) != 1 {
		t.Fatalf("hook events = %d want 1", len(hook.events))
	}
	if hook.events[0].Action != "URL_UPDATED" {
		t.Fatalf("hook event action = %q want URL_UPDATED", hook.events[0].Action)
	}
	if len(hook.events[0].URLs) != 1 || hook.events[0].URLs[0] != "https://example.com/" {
		t.Fatalf("hook event URLs = %#v want root URL", hook.events[0].URLs)
	}
	if got.Downstream == nil {
		t.Fatal("Downstream unexpectedly nil")
	}

	jsdelivrPath := filepath.Join(root, "themes", "LoveIt", "assets", "data", "cdn", "jsdelivr.yml")
	raw, err := os.ReadFile(jsdelivrPath)
	if err != nil {
		t.Fatalf("ReadFile(jsdelivr) error = %v", err)
	}
	if !strings.Contains(string(raw), "@4.1.2/") {
		t.Fatalf("jsdelivr.yml not updated: %s", raw)
	}
	sriPath := filepath.Join(root, "data", "sri.yaml")
	raw, err = os.ReadFile(sriPath)
	if err != nil {
		t.Fatalf("ReadFile(sri) error = %v", err)
	}
	if !strings.Contains(string(raw), "@4.1.2/") {
		t.Fatalf("sri.yaml not updated: %s", raw)
	}
}

func TestServiceCheckDryRunSkipsAutoFixAndDownstream(t *testing.T) {
	root := writeSriFixture(t, sriFixture{
		PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`,
		SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css",
		SRIHashBody:     []byte("animate-old"),
		JSDelivrVersion: "4.1.1",
	})
	hook := &fakeHookProcessor{}
	svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "data.jsdelivr.com":
			return jsonResponse(`{"tags":{"latest":"4.1.2"}}`), nil
		case req.URL.String() == "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css":
			return bodyResponse("animate-old"), nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	})}, hook, nil)

	got, err := svc.Check(context.Background(), Request{AutoFix: boolPtr(true), DryRun: boolPtr(true)})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if got.Report.AutoFix.Ran {
		t.Fatal("AutoFix.Ran unexpectedly true")
	}
	if !got.Report.AutoFix.Skipped {
		t.Fatal("AutoFix.Skipped unexpectedly false")
	}
	if len(hook.events) != 0 {
		t.Fatalf("hook events = %d want 0", len(hook.events))
	}
}

func TestServiceDetectsSRIProblemsAndHostGuards(t *testing.T) {
	t.Run("missing sri entry", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`,
			JSDelivrVersion: "4.1.1",
			SkipSRIEntry:    true,
		})
		svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "data.jsdelivr.com" {
				return jsonResponse(`{"tags":{"latest":"4.1.1"}}`), nil
			}
			return bodyResponse("animate-ok"), nil
		})}, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if got.Report.Summary != "WARN" {
			t.Fatalf("Summary = %q want WARN", got.Report.Summary)
		}
		if !strings.Contains(strings.ToUpper(strings.Join(got.Report.Diagnostic.Other, "\n")), "MISSING SRI ENTRY") {
			t.Fatalf("Other = %#v want missing sri entry", got.Report.Diagnostic.Other)
		}
	})

	t.Run("sri mismatch", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`,
			SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css",
			SRIHashBody:     []byte("wrong-body"),
			JSDelivrVersion: "4.1.1",
		})
		svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "data.jsdelivr.com" {
				return jsonResponse(`{"tags":{"latest":"4.1.1"}}`), nil
			}
			return bodyResponse("animate-live"), nil
		})}, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if len(got.Report.Diagnostic.HashMismatch) == 0 {
			t.Fatal("HashMismatch unexpectedly empty")
		}
	})

	t.Run("disallowed host", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://evil.example.invalid/npm/foo@1.0.0/x.js"></script>`,
			JSDelivrVersion: "1.0.0",
			SkipSRIEntry:    true,
		})
		svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(`{"tags":{"latest":"1.0.0"}}`), nil
		})}, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "host not allowlisted") {
			t.Fatalf("Other = %#v want host not allowlisted", got.Report.Diagnostic.Other)
		}
	})
}

func TestServiceGuardsAgainstLargeFilesSymlinksAndMalformedInput(t *testing.T) {
	t.Run("too large file", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   strings.Repeat("x", 128),
			JSDelivrVersion: "1.0.0",
			SkipSRIEntry:    true,
		})
		svc := newTestService(t, root, nil, nil, nil)
		svc.cfg.MaxFileBytes = 16
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "too large") {
			t.Fatalf("Other = %#v want too large", got.Report.Diagnostic.Other)
		}
	})

	t.Run("symlink escape", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`,
			JSDelivrVersion: "4.1.1",
			SkipSRIEntry:    true,
		})
		outside := filepath.Join(t.TempDir(), "outside.html")
		if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
			t.Fatalf("write outside: %v", err)
		}
		link := filepath.Join(root, "public", "linked.html")
		if err := os.Symlink(outside, link); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		svc := newTestService(t, root, nil, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "symlink") {
			t.Fatalf("Other = %#v want symlink", got.Report.Diagnostic.Other)
		}
	})

	t.Run("malformed yaml", func(t *testing.T) {
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, "themes", "LoveIt", "assets", "data", "cdn"), "jsdelivr.yml", []byte("prefix: ["))
		mustWrite(t, filepath.Join(root, "data"), "sri.yaml", []byte("not: [valid"))
		mustWrite(t, filepath.Join(root, "public"), "index.html", []byte(`<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`))
		svc := newTestService(t, root, nil, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if got.Report.Summary != "WARN" {
			t.Fatalf("Summary = %q want WARN", got.Report.Summary)
		}
		if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "yaml") {
			t.Fatalf("Other = %#v want yaml error", got.Report.Diagnostic.Other)
		}
	})
}

func TestServiceRejectsScanRootsOutsideHugoRoot(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, nil, nil, nil)
	svc.cfg.ScanRoots = []string{"../escape"}
	got, err := svc.Check(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "path traversal") {
		t.Fatalf("Other = %#v want path traversal", got.Report.Diagnostic.Other)
	}
}

func newTestService(t *testing.T, root string, client *http.Client, hook *fakeHookProcessor, build *fakeBuilder) *Service {
	t.Helper()
	svc, err := NewService(Config{
		Enabled:           true,
		TriggerHooksOnFix: true,
		DryRunDefault:     true,
		MaxFileBytes:      1 << 20,
		MaxFiles:          256,
		HugoRoot:          root,
		SiteBaseURL:       "https://example.com",
		AllowedCDNHosts:   []string{"cdn.jsdelivr.net", "fastly.jsdelivr.net", "data.jsdelivr.com"},
		ScanRoots:         []string{"public", "themes/LoveIt/layouts", "content"},
	}, WithHTTPClient(client), WithHooks(hook), WithBuilder(build))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

type sriFixture struct {
	PublicContent   string
	LayoutContent   string
	SRIHashURL      string
	SRIHashBody     []byte
	JSDelivrVersion string
	SkipSRIEntry    bool
}

func boolPtr(v bool) *bool {
	return &v
}

func writeSriFixture(t *testing.T, fx sriFixture) string {
	t.Helper()
	root := t.TempDir()
	jsdelivr := `prefix:
  libFiles: https://cdn.jsdelivr.net/npm/
libFiles:
  animateCSS: animate.css@4.1.1/animate.min.css
`
	if fx.JSDelivrVersion != "" {
		jsdelivr = strings.ReplaceAll(jsdelivr, "4.1.1", fx.JSDelivrVersion)
	}
	mustWrite(t, filepath.Join(root, "themes", "LoveIt", "assets", "data", "cdn"), "jsdelivr.yml", []byte(jsdelivr))

	if !fx.SkipSRIEntry {
		hash := sriHash(fx.SRIHashBody)
		line := `"` + fx.SRIHashURL + `": "` + hash + `"` + "\n"
		mustWrite(t, filepath.Join(root, "data"), "sri.yaml", []byte(line))
	} else {
		mustWrite(t, filepath.Join(root, "data"), "sri.yaml", []byte{})
	}
	mustWrite(t, filepath.Join(root, "public"), "index.html", []byte(fx.PublicContent))
	if fx.LayoutContent != "" {
		mustWrite(t, filepath.Join(root, "themes", "LoveIt", "layouts", "_partials"), "head.html", []byte(fx.LayoutContent))
	}
	return root
}

func mustWrite(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s/%s) error = %v", dir, name, err)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func bodyResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func sriHash(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
}
