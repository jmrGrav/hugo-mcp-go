package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

const (
	DefaultIndexNowEndpoint = "https://api.indexnow.org/indexnow"
	DefaultHooksMaxRetries  = 5
)

type Config struct {
	PostBuildHooksEnabled            bool
	CloudflarePurgeEnabled           bool
	CloudflareAllowPurgeEverything   bool
	CloudflareZoneID                 string
	CloudflareTokenFile              string
	GoogleIndexingEnabled            bool
	GoogleIndexingServiceAccountFile string
	IndexNowEnabled                  bool
	IndexNowKeyFile                  string
	IndexNowEndpoint                 string
	SiteBaseURL                      string
	HooksDB                          string
	HooksDryRun                      bool
	HooksMaxRetries                  int
	HooksAdminEnabled                bool
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		PostBuildHooksEnabled:            parseBoolEnv("HUGO_POST_BUILD_HOOKS_ENABLED", true),
		CloudflarePurgeEnabled:           parseBoolEnv("HUGO_CLOUDFLARE_PURGE_ENABLED", false),
		CloudflareAllowPurgeEverything:   parseBoolEnv("HUGO_CLOUDFLARE_ALLOW_PURGE_EVERYTHING", false),
		CloudflareZoneID:                 strings.TrimSpace(os.Getenv("HUGO_CLOUDFLARE_ZONE_ID")),
		CloudflareTokenFile:              strings.TrimSpace(os.Getenv("HUGO_CLOUDFLARE_TOKEN_FILE")),
		GoogleIndexingEnabled:            parseBoolEnv("HUGO_GOOGLE_INDEXING_ENABLED", false),
		GoogleIndexingServiceAccountFile: strings.TrimSpace(os.Getenv("HUGO_GOOGLE_INDEXING_SERVICE_ACCOUNT_FILE")),
		IndexNowEnabled:                  parseBoolEnv("HUGO_INDEXNOW_ENABLED", false),
		IndexNowKeyFile:                  strings.TrimSpace(os.Getenv("HUGO_INDEXNOW_KEY_FILE")),
		IndexNowEndpoint:                 envOrDefault("HUGO_INDEXNOW_ENDPOINT", DefaultIndexNowEndpoint),
		SiteBaseURL:                      strings.TrimSpace(os.Getenv("HUGO_SITE_BASE_URL")),
		HooksDB:                          envOrDefault("HUGO_HOOKS_DB", filepath.Join(os.TempDir(), "hugo-mcp-go-hooks.db")),
		HooksDryRun:                      parseBoolEnv("HUGO_HOOKS_DRY_RUN", true),
		HooksMaxRetries:                  parseIntEnv("HUGO_HOOKS_MAX_RETRIES", DefaultHooksMaxRetries),
		HooksAdminEnabled:                parseBoolEnv("HUGO_HOOKS_ADMIN_ENABLED", false),
	}
	if cfg.HooksMaxRetries <= 0 {
		return Config{}, fmt.Errorf("invalid HUGO_HOOKS_MAX_RETRIES: must be positive")
	}
	return cfg, nil
}

func LoadSecretFile(path, allowedDir string) ([]byte, error) {
	path = strings.TrimSpace(path)
	allowedDir = strings.TrimSpace(allowedDir)
	if path == "" {
		return nil, errors.New("missing secret file path")
	}
	if allowedDir == "" {
		return nil, errors.New("missing allowed secret directory")
	}
	allowedRoot, err := pathguard.CanonicalDir(allowedDir)
	if err != nil {
		return nil, fmt.Errorf("invalid allowed secret directory: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !pathguard.WithinRoot(allowedRoot, absPath) {
		return nil, fmt.Errorf("secret file must be inside %s", allowedRoot)
	}
	info, err := os.Lstat(absPath)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("secret file must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("secret file must be a regular file")
	}
	perm := info.Mode().Perm() & 0o777
	if perm != 0o600 && perm != 0o640 {
		return nil, fmt.Errorf("secret file permissions too permissive: %04o", perm)
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("secret file is empty")
	}
	return raw, nil
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

func envOrDefault(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}
