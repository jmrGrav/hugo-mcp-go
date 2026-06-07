package sri

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceConstructorDisabledAndNilBranches(t *testing.T) {
	root := t.TempDir()
	svc, err := NewService(Config{
		Enabled:         false,
		DryRunDefault:   true,
		HugoRoot:        root,
		SiteBaseURL:     "https://example.com",
		AllowedCDNHosts: []string{"cdn.jsdelivr.net"},
		ScanRoots:       []string{"public"},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc.httpClient == nil {
		t.Fatal("NewService() did not initialize an HTTP client")
	}
	got, err := svc.Check(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if got.Report.Summary != "DISABLED" {
		t.Fatalf("Summary = %q want DISABLED", got.Report.Summary)
	}
	if got.ExitCode != 2 {
		t.Fatalf("ExitCode = %d want 2", got.ExitCode)
	}

	var nilSvc *Service
	if _, err := nilSvc.Check(context.Background(), Request{}); err == nil {
		t.Fatal("nil service expected error")
	}
}

func TestServiceFetchParseAndFindHelpers(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "themes", "LoveIt", "assets", "data", "cdn"), "jsdelivr.yml", []byte(`libFiles:
  animateCSS: animate.css@4.1.1/animate.min.css
`))
	mustWrite(t, filepath.Join(root, "data"), "sri.yaml", []byte(`"https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css": "sha256-abc"
`))
	mustWrite(t, filepath.Join(root, "public"), "index.html", []byte(`<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`))

	svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.String(), "data.jsdelivr.com"):
			return bodyResponse(`{`), nil
		case strings.Contains(req.URL.String(), "boom"):
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("boom")), Header: make(http.Header)}, nil
		default:
			return bodyResponse(`{"tags":{"latest":"4.1.2"}}`), nil
		}
	})}, nil, nil)

	if _, err := svc.fetchBytes(context.Background(), "https://example.com/boom"); err == nil {
		t.Fatal("fetchBytes() expected HTTP status error")
	}
	if _, err := svc.fetchLatestVersion(context.Background(), "animate.css"); err == nil {
		t.Fatal("fetchLatestVersion() expected JSON error")
	}

	cdnPath, sriPath, warnings := svc.locateDataFiles()
	if len(warnings) != 0 {
		t.Fatalf("locateDataFiles() warnings = %#v", warnings)
	}
	if !strings.HasSuffix(cdnPath, filepath.FromSlash("assets/data/cdn/jsdelivr.yml")) {
		t.Fatalf("cdn path = %q", cdnPath)
	}
	if !strings.HasSuffix(sriPath, filepath.FromSlash("data/sri.yaml")) {
		t.Fatalf("sri path = %q", sriPath)
	}

	cdnEntries, warnings, err := parseCDNConfig(cdnPath)
	if err != nil {
		t.Fatalf("parseCDNConfig() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("parseCDNConfig() warnings = %#v", warnings)
	}
	if len(cdnEntries) != 1 {
		t.Fatalf("parseCDNConfig() entries = %#v", cdnEntries)
	}

	sriEntries, warnings, err := parseSRIConfig(sriPath)
	if err != nil {
		t.Fatalf("parseSRIConfig() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("parseSRIConfig() warnings = %#v", warnings)
	}
	if len(sriEntries) != 1 {
		t.Fatalf("parseSRIConfig() entries = %#v", sriEntries)
	}

	if _, _, err := parseCDNConfig(filepath.Join(root, "missing.yml")); err == nil {
		t.Fatal("parseCDNConfig(missing) expected error")
	}
	if _, _, err := parseSRIConfig(filepath.Join(root, "missing.yaml")); err == nil {
		t.Fatal("parseSRIConfig(missing) expected error")
	}

	found, err := findFileBySuffix(root, filepath.FromSlash("assets/data/cdn/jsdelivr.yml"))
	if err != nil || found == "" {
		t.Fatalf("findFileBySuffix() error=%v found=%q", err, found)
	}
	if _, err := findFileBySuffix(root, filepath.FromSlash("missing/file.txt")); err == nil {
		t.Fatal("findFileBySuffix() expected missing file error")
	}
}

func TestServiceUtilityBranchesAndRollback(t *testing.T) {
	if sameMajor("4.1.1", "4.1.2") != true {
		t.Fatal("sameMajor() expected true")
	}
	if compareVersions("4.1.1", "4.1.2") >= 0 || compareVersions("4.1.2", "4.1.1") <= 0 || compareVersions("4.1.1", "4.1.1") != 0 {
		t.Fatal("compareVersions() ordering failed")
	}
	if majorPart("4.1.1") != "4" || majorPart("4") != "4" {
		t.Fatal("majorPart() unexpected result")
	}
	if got := splitVersionParts("1.2.3"); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("splitVersionParts() = %#v", got)
	}
	if !hostAllowed("cdn.jsdelivr.net", []string{"cdn.jsdelivr.net"}) || hostAllowed("evil.example", []string{"cdn.jsdelivr.net"}) {
		t.Fatal("hostAllowed() branch failed")
	}
	if !isTextLike("index.html") || isTextLike("archive.bin") {
		t.Fatal("isTextLike() branch failed")
	}
	if got := joinBaseURL("https://example.com/", "/posts/x/"); got != "https://example.com/posts/x" {
		t.Fatalf("joinBaseURL() = %q", got)
	}
	if got := redactScanError(errors.New("boom")); got != "boom" {
		t.Fatalf("redactScanError() = %q", got)
	}

	root := writeSriFixture(t, sriFixture{
		PublicContent:   `<script src="https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css"></script>`,
		SRIHashURL:      "https://cdn.jsdelivr.net/npm/animate.css@4.1.1/animate.min.css",
		SRIHashBody:     []byte("animate-old"),
		JSDelivrVersion: "4.1.1",
	})
	builder := &fakeBuilder{
		err: errors.New("build failed"),
	}
	svc := newTestService(t, root, &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "data.jsdelivr.com":
			return jsonResponse(`{"tags":{"latest":"4.1.2"}}`), nil
		case strings.Contains(req.URL.String(), "4.1.1"):
			return bodyResponse("animate-old"), nil
		case strings.Contains(req.URL.String(), "4.1.2"):
			return bodyResponse("animate-new"), nil
		default:
			return bodyResponse("animate-old"), nil
		}
	})}, &fakeHookProcessor{}, builder)

	got, err := svc.Check(context.Background(), Request{AutoFix: boolPtr(true), DryRun: boolPtr(false)})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(builder.calls) != 1 {
		t.Fatalf("Build() calls = %d want 1", len(builder.calls))
	}
	if !strings.Contains(strings.Join(got.Report.Diagnostic.Other, "\n"), "autofix build failed") {
		t.Fatalf("Other = %#v want autofix build failed", got.Report.Diagnostic.Other)
	}
	raw, err := os.ReadFile(filepath.Join(root, "themes", "LoveIt", "assets", "data", "cdn", "jsdelivr.yml"))
	if err != nil {
		t.Fatalf("ReadFile(cdn) error = %v", err)
	}
	if !strings.Contains(string(raw), "@4.1.1/") {
		t.Fatalf("cdn file unexpectedly changed: %s", raw)
	}
	raw, err = os.ReadFile(filepath.Join(root, "data", "sri.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(sri) error = %v", err)
	}
	if !strings.Contains(string(raw), "@4.1.1/") {
		t.Fatalf("sri file unexpectedly changed: %s", raw)
	}
}
