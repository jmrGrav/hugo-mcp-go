package sri

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFromEnvParsesAndNormalizes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HUGO_SRI_CHECK_ENABLED", "false")
	t.Setenv("HUGO_SRI_TRIGGER_HOOKS_ON_FIX", "false")
	t.Setenv("HUGO_SRI_DRY_RUN_DEFAULT", "false")
	t.Setenv("HUGO_SRI_MAX_FILE_BYTES", "2048")
	t.Setenv("HUGO_SRI_MAX_FILES", "12")
	t.Setenv("HUGO_SRI_SCAN_ROOTS", "public, public ,layouts")
	t.Setenv("HUGO_SRI_ALLOWED_CDN_HOSTS", "CDN.JSDELIVR.NET, cdn.jsdelivr.net, ")

	cfg, err := LoadConfigFromEnv(root, "https://example.com/")
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.Enabled || cfg.TriggerHooksOnFix || cfg.DryRunDefault {
		t.Fatalf("unexpected booleans: %+v", cfg)
	}
	if cfg.MaxFileBytes != 2048 || cfg.MaxFiles != 12 {
		t.Fatalf("unexpected limits: %+v", cfg)
	}
	if got := strings.Join(cfg.ScanRoots, ","); got != "public,layouts" {
		t.Fatalf("ScanRoots = %q want public,layouts", got)
	}
	if got := strings.Join(cfg.AllowedCDNHosts, ","); got != "cdn.jsdelivr.net" {
		t.Fatalf("AllowedCDNHosts = %q want cdn.jsdelivr.net", got)
	}
	if cfg.HugoRoot != root {
		t.Fatalf("HugoRoot = %q want %q", cfg.HugoRoot, root)
	}
	if cfg.SiteBaseURL != "https://example.com" {
		t.Fatalf("SiteBaseURL = %q want trimmed base URL", cfg.SiteBaseURL)
	}
}

func TestConfigValidateAppliesDefaultsAndRejectsInvalidRoots(t *testing.T) {
	root := t.TempDir()
	cfg, err := (Config{HugoRoot: root}).Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.MaxFileBytes != DefaultMaxFileBytes || cfg.MaxFiles != DefaultMaxFiles {
		t.Fatalf("Validate() did not apply defaults: %+v", cfg)
	}
	if len(cfg.ScanRoots) != len(defaultScanRoots) || len(cfg.AllowedCDNHosts) != len(defaultAllowedHosts) {
		t.Fatalf("Validate() did not restore defaults: %+v", cfg)
	}

	if _, err := (Config{HugoRoot: filepath.Join("..", "escape")}).Validate(); err == nil {
		t.Fatal("Validate() expected invalid root error")
	}
	if _, err := (Config{}).Validate(); err == nil {
		t.Fatal("Validate() expected missing root error")
	}
}

func TestConfigParsingHelpersAndDeduping(t *testing.T) {
	t.Setenv("SRI_BOOL", "not-a-bool")
	if got := parseBoolEnv("SRI_BOOL", true); !got {
		t.Fatalf("parseBoolEnv fallback = %v want true", got)
	}
	t.Setenv("SRI_INT", "not-a-number")
	if got := parseIntEnv("SRI_INT", 42); got != 42 {
		t.Fatalf("parseIntEnv fallback = %d want 42", got)
	}
	t.Setenv("SRI_INT64", "not-a-number")
	if got := parseInt64Env("SRI_INT64", 64); got != 64 {
		t.Fatalf("parseInt64Env fallback = %d want 64", got)
	}
	t.Setenv("SRI_CSV", " one, one, two ,, ")
	if got := parseCSVEnv("SRI_CSV", []string{"fallback"}); strings.Join(got, ",") != "one,one,two" {
		t.Fatalf("parseCSVEnv = %#v", got)
	}
	t.Setenv("SRI_CSV_EMPTY", " , , ")
	if got := parseCSVEnv("SRI_CSV_EMPTY", []string{"fallback"}); strings.Join(got, ",") != "fallback" {
		t.Fatalf("parseCSVEnv empty fallback = %#v", got)
	}

	if got := dedupeCSV([]string{"a", "a", "b", "", " c "}); strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("dedupeCSV = %#v", got)
	}
	if got := normalizeHosts([]string{" CDN.JsDelivr.Net ", "cdn.jsdelivr.net", "data.jsdelivr.com"}); strings.Join(got, ",") != "cdn.jsdelivr.net,data.jsdelivr.com" {
		t.Fatalf("normalizeHosts = %#v", got)
	}
}
