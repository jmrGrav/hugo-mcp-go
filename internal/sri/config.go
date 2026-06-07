package sri

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

const (
	DefaultMaxFileBytes = 1 << 20
	DefaultMaxFiles     = 2000
)

var (
	defaultScanRoots = []string{
		"public",
		"content",
		"layouts",
		"assets",
		"themes/LoveIt/layouts",
	}
	defaultAllowedHosts = []string{
		"cdn.jsdelivr.net",
		"fastly.jsdelivr.net",
		"data.jsdelivr.com",
	}
)

type Config struct {
	Enabled           bool
	TriggerHooksOnFix bool
	DryRunDefault     bool
	MaxFileBytes      int64
	MaxFiles          int
	ScanRoots         []string
	AllowedCDNHosts   []string
	HugoRoot          string
	SiteBaseURL       string
}

func LoadConfigFromEnv(hugoRoot, siteBaseURL string) (Config, error) {
	cfg := Config{
		Enabled:           parseBoolEnv("HUGO_SRI_CHECK_ENABLED", true),
		TriggerHooksOnFix: parseBoolEnv("HUGO_SRI_TRIGGER_HOOKS_ON_FIX", true),
		DryRunDefault:     parseBoolEnv("HUGO_SRI_DRY_RUN_DEFAULT", true),
		MaxFileBytes:      parseInt64Env("HUGO_SRI_MAX_FILE_BYTES", DefaultMaxFileBytes),
		MaxFiles:          parseIntEnv("HUGO_SRI_MAX_FILES", DefaultMaxFiles),
		ScanRoots:         parseCSVEnv("HUGO_SRI_SCAN_ROOTS", defaultScanRoots),
		AllowedCDNHosts:   parseCSVEnv("HUGO_SRI_ALLOWED_CDN_HOSTS", defaultAllowedHosts),
		HugoRoot:          strings.TrimSpace(hugoRoot),
		SiteBaseURL:       strings.TrimSpace(siteBaseURL),
	}
	return cfg.Validate()
}

func (c Config) Validate() (Config, error) {
	if c.HugoRoot == "" {
		return Config{}, fmt.Errorf("missing Hugo root")
	}
	root, err := pathguard.CanonicalDir(c.HugoRoot)
	if err != nil {
		return Config{}, fmt.Errorf("invalid Hugo root: %w", err)
	}
	if c.MaxFileBytes <= 0 {
		c.MaxFileBytes = DefaultMaxFileBytes
	}
	if c.MaxFiles <= 0 {
		c.MaxFiles = DefaultMaxFiles
	}
	c.ScanRoots = dedupeCSV(c.ScanRoots)
	if len(c.ScanRoots) == 0 {
		c.ScanRoots = append([]string(nil), defaultScanRoots...)
	}
	c.AllowedCDNHosts = normalizeHosts(c.AllowedCDNHosts)
	if len(c.AllowedCDNHosts) == 0 {
		c.AllowedCDNHosts = append([]string(nil), defaultAllowedHosts...)
	}
	c.HugoRoot = root
	c.SiteBaseURL = strings.TrimRight(strings.TrimSpace(c.SiteBaseURL), "/")
	return c, nil
}

func parseBoolEnv(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseIntEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseInt64Env(name string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

func parseCSVEnv(name string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}

func dedupeCSV(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeHosts(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
