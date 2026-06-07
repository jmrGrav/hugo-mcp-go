package sri

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
)

func TestServiceVersionEvaluationBranches(t *testing.T) {
	t.Run("major outdated", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css"></script>`,
			SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css",
			SRIHashBody:     []byte("animate-old"),
			JSDelivrVersion: "1.0.0",
		})
		svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Host == "data.jsdelivr.com":
				return jsonResponse(`{"tags":{"latest":"2.0.0"}}`), nil
			default:
				return bodyResponse("animate-old"), nil
			}
		})}, nil, nil)

		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if got.Report.Summary != "WARN" {
			t.Fatalf("Summary = %q want WARN", got.Report.Summary)
		}
		if !strings.Contains(strings.Join(got.Report.Diagnostic.MajorOutdated, "\n"), "OUTDATED MAJOR") {
			t.Fatalf("MajorOutdated = %#v want major warning", got.Report.Diagnostic.MajorOutdated)
		}
	})

	t.Run("local version newer than latest skips warning", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@2.0.0/animate.min.css"></script>`,
			SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@2.0.0/animate.min.css",
			SRIHashBody:     []byte("animate-old"),
			JSDelivrVersion: "2.0.0",
		})
		svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Host == "data.jsdelivr.com":
				return jsonResponse(`{"tags":{"latest":"1.0.0"}}`), nil
			default:
				return bodyResponse("animate-old"), nil
			}
		})}, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if got.Report.Summary != "OK" {
			t.Fatalf("Summary = %q want OK", got.Report.Summary)
		}
	})

	t.Run("api fail", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css"></script>`,
			SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css",
			SRIHashBody:     []byte("animate-old"),
			JSDelivrVersion: "1.0.0",
		})
		svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "data.jsdelivr.com" {
				return nil, errors.New("boom")
			}
			return bodyResponse("animate-old"), nil
		})}, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "API FAIL") {
			t.Fatalf("Other = %#v want API FAIL", got.Report.Diagnostic.Other)
		}
	})

	t.Run("fetch fail", func(t *testing.T) {
		root := writeSriFixture(t, sriFixture{
			PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css"></script>`,
			SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css",
			SRIHashBody:     []byte("animate-old"),
			JSDelivrVersion: "1.0.0",
		})
		svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Host == "data.jsdelivr.com":
				return jsonResponse(`{"tags":{"latest":"1.0.0"}}`), nil
			default:
				return nil, errors.New("boom")
			}
		})}, nil, nil)
		got, err := svc.Check(context.Background(), Request{})
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "FETCH FAIL") {
			t.Fatalf("Other = %#v want FETCH FAIL", got.Report.Diagnostic.Other)
		}
	})
}

func TestServiceScanRootsWarningsAndLimits(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "not-a-dir"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write not-a-dir: %v", err)
	}
	mustWrite(t, filepath.Join(root, "public"), "index.html", []byte(`<script src="https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css"></script>`))
	mustWrite(t, filepath.Join(root, "public"), "second.html", []byte(`<script src="https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css"></script>`))
	mustWrite(t, filepath.Join(root, "public"), "blob.bin", []byte{0x00, 0x01, 0x02})

	svc := newTestService(t, root, nil, nil, nil)
	svc.cfg.ScanRoots = []string{"public", "not-a-dir"}
	svc.cfg.MaxFiles = 2
	state := newRunState()
	warnings := svc.scanRoots(context.Background(), state)
	if !strings.Contains(strings.Join(warnings, "\n"), "not a directory") {
		t.Fatalf("warnings = %#v want not a directory", warnings)
	}
	if !strings.Contains(strings.Join(warnings, "\n"), "scan file limit reached") {
		t.Fatalf("warnings = %#v want file limit reached", warnings)
	}
	if len(state.ActiveRefs) == 0 {
		t.Fatal("expected active refs to be discovered before limit")
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := svc.scanRoots(cancelCtx, newRunState()); len(got) == 0 {
		t.Fatal("expected cancelled scan to return warnings")
	}
}

func TestServiceHelperEdgesAndFetchByteFailures(t *testing.T) {
	root := t.TempDir()
	svc, err := NewService(Config{
		Enabled:           true,
		TriggerHooksOnFix: true,
		DryRunDefault:     true,
		HugoRoot:          root,
		SiteBaseURL:       "",
		AllowedCDNHosts:   []string{"cdn.jsdelivr.net"},
		ScanRoots:         []string{"public"},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	svc2, err := NewService(Config{
		Enabled:         true,
		DryRunDefault:   true,
		HugoRoot:        root,
		AllowedCDNHosts: []string{"cdn.jsdelivr.net"},
		ScanRoots:       []string{"public"},
	}, WithHTTPClient(nil))
	if err != nil {
		t.Fatalf("NewService(nil client) error = %v", err)
	}
	if svc2.httpClient == nil {
		t.Fatal("NewService(nil client) did not restore default client")
	}
	svc.httpClient = nil
	if _, err := svc.fetchBytes(context.Background(), "https://example.com"); err == nil {
		t.Fatal("fetchBytes() expected nil client error")
	}

	if got := majorPart(""); got != "" {
		t.Fatalf("majorPart(\"\") = %q want empty", got)
	}
	if got := joinBaseURL("", "/"); got != "" {
		t.Fatalf("joinBaseURL(empty) = %q want empty", got)
	}
	if got := redactScanError(nil); got != "" {
		t.Fatalf("redactScanError(nil) = %q want empty", got)
	}
	if got := parseBoolEnv("MISSING_BOOL", true); !got {
		t.Fatalf("parseBoolEnv missing = %v want true", got)
	}
	if got := parseIntEnv("MISSING_INT", 7); got != 7 {
		t.Fatalf("parseIntEnv missing = %d want 7", got)
	}
	if got := parseInt64Env("MISSING_INT64", 9); got != 9 {
		t.Fatalf("parseInt64Env missing = %d want 9", got)
	}
	if got := parseCSVEnv("MISSING_CSV", nil); len(got) != 0 {
		t.Fatalf("parseCSVEnv missing = %#v want empty", got)
	}

	finished := &Result{Report: Report{Summary: "DISABLED"}, ExitCode: 2}
	finalizeResult(finished)
	if finished.Report.Exit != 2 || finished.Success {
		t.Fatalf("finalizeResult(DISABLED) = %#v", finished)
	}
	finalizeResult(nil)
}

func TestServiceEvaluateSRIAndAutofixBranchEdges(t *testing.T) {
	root := writeSriFixture(t, sriFixture{
		PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css"></script>`,
		SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@1.0.0/animate.min.css",
		SRIHashBody:     []byte("animate-old"),
		JSDelivrVersion: "1.0.0",
	})
	svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "data.jsdelivr.com":
			return jsonResponse(`{"tags":{"latest":"1.0.0"}}`), nil
		case strings.Contains(req.URL.String(), "animate.min.css"):
			return bodyResponse("animate-old"), nil
		default:
			return bodyResponse("animate-old"), nil
		}
	})}, nil, &fakeBuilder{result: mutations.BuildResult{Status: "built", Deploy: "DEPLOY_SKIPPED"}})
	svc.cfg.TriggerHooksOnFix = false
	svc.cfg.SiteBaseURL = ""

	got, err := svc.Check(context.Background(), Request{AutoFix: boolPtr(true), DryRun: boolPtr(false)})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if got.Downstream != nil {
		t.Fatalf("Downstream = %#v want nil when hooks are disabled", got.Downstream)
	}

	state := newRunState()
	state.ActiveRefs["pkg"] = activeRef{URL: "https://cdn.jsdelivr.net/npm/x@1.0.0/x.js", Package: "x"}
	svc.evaluateSRI(context.Background(), state, map[string]string{
		"http://example.com/x.js":   "sha256-abc",
		"https://evil.example/x.js": "sha256-abc",
		"https://%zz":               "sha256-abc",
	})
	joined := strings.Join(state.Other, "\n")
	if !strings.Contains(joined, "host not allowlisted") {
		t.Fatalf("Other = %#v want host guard warning", state.Other)
	}
	if !strings.Contains(joined, "invalid SRI url") {
		t.Fatalf("Other = %#v want invalid SRI url warning", state.Other)
	}
}
