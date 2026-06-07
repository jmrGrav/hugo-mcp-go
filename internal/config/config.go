package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

const (
	DefaultMaxRequestBytes  = 1 << 20
	DefaultMaxToolArgsBytes = 1 << 18
	DefaultMaxPageBytes     = 1 << 20
	DefaultMaxAssetBytes    = 25 << 20
	DefaultMaxListPages     = 500
	DefaultMaxListAssets    = 500
)

type Config struct {
	HugoRoot         string
	ContentRoot      string
	StaticRoot       string
	MaxRequestBytes  int64
	MaxToolArgsBytes int64
	MaxPageBytes     int64
	MaxAssetBytes    int64
	MaxListPages     int
	MaxListAssets    int
}

func LoadFromEnv() (Config, error) {
	maxRequestBytes, err := parseInt64Env("HUGO_MAX_REQUEST_BYTES", DefaultMaxRequestBytes)
	if err != nil {
		return Config{}, err
	}
	maxToolArgsBytes, err := parseInt64Env("HUGO_MAX_TOOL_ARGS_BYTES", DefaultMaxToolArgsBytes)
	if err != nil {
		return Config{}, err
	}
	maxPageBytes, err := parseInt64Env("HUGO_MAX_PAGE_BYTES", DefaultMaxPageBytes)
	if err != nil {
		return Config{}, err
	}
	maxAssetBytes, err := parseInt64Env("HUGO_MAX_ASSET_BYTES", DefaultMaxAssetBytes)
	if err != nil {
		return Config{}, err
	}
	maxListPages, err := parseIntEnv("HUGO_MAX_LIST_PAGES", DefaultMaxListPages)
	if err != nil {
		return Config{}, err
	}
	maxListAssets, err := parseIntEnv("HUGO_MAX_LIST_ASSETS", DefaultMaxListAssets)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HugoRoot:         strings.TrimSpace(os.Getenv("HUGO_ROOT")),
		ContentRoot:      strings.TrimSpace(os.Getenv("HUGO_CONTENT_ROOT")),
		StaticRoot:       strings.TrimSpace(os.Getenv("HUGO_STATIC_ROOT")),
		MaxRequestBytes:  maxRequestBytes,
		MaxToolArgsBytes: maxToolArgsBytes,
		MaxPageBytes:     maxPageBytes,
		MaxAssetBytes:    maxAssetBytes,
		MaxListPages:     maxListPages,
		MaxListAssets:    maxListAssets,
	}
	return cfg.Validate()
}

func (c Config) Validate() (Config, error) {
	if c.HugoRoot == "" {
		return Config{}, fmt.Errorf("missing required env HUGO_ROOT")
	}
	if c.ContentRoot == "" {
		return Config{}, fmt.Errorf("missing required env HUGO_CONTENT_ROOT")
	}
	if c.StaticRoot == "" {
		return Config{}, fmt.Errorf("missing required env HUGO_STATIC_ROOT")
	}

	hugoRoot, err := pathguard.CanonicalDir(c.HugoRoot)
	if err != nil {
		return Config{}, fmt.Errorf("invalid HUGO_ROOT: %w", err)
	}
	contentRoot, err := pathguard.CanonicalDir(c.ContentRoot)
	if err != nil {
		return Config{}, fmt.Errorf("invalid HUGO_CONTENT_ROOT: %w", err)
	}
	staticRoot, err := pathguard.CanonicalDir(c.StaticRoot)
	if err != nil {
		return Config{}, fmt.Errorf("invalid HUGO_STATIC_ROOT: %w", err)
	}

	if !pathguard.WithinRoot(hugoRoot, contentRoot) {
		return Config{}, fmt.Errorf("HUGO_CONTENT_ROOT must be inside HUGO_ROOT")
	}
	if !pathguard.WithinRoot(hugoRoot, staticRoot) {
		return Config{}, fmt.Errorf("HUGO_STATIC_ROOT must be inside HUGO_ROOT")
	}

	if c.MaxRequestBytes <= 0 || c.MaxToolArgsBytes <= 0 || c.MaxPageBytes <= 0 || c.MaxAssetBytes <= 0 {
		return Config{}, fmt.Errorf("size limits must be positive")
	}
	if c.MaxListPages <= 0 || c.MaxListAssets <= 0 {
		return Config{}, fmt.Errorf("list limits must be positive")
	}

	c.HugoRoot = hugoRoot
	c.ContentRoot = contentRoot
	c.StaticRoot = staticRoot
	return c, nil
}

func parseIntEnv(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}

func parseInt64Env(name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}

func FromRoots(hugoRoot, contentRoot, staticRoot string) Config {
	return Config{
		HugoRoot:         hugoRoot,
		ContentRoot:      contentRoot,
		StaticRoot:       staticRoot,
		MaxRequestBytes:  DefaultMaxRequestBytes,
		MaxToolArgsBytes: DefaultMaxToolArgsBytes,
		MaxPageBytes:     DefaultMaxPageBytes,
		MaxAssetBytes:    DefaultMaxAssetBytes,
		MaxListPages:     DefaultMaxListPages,
		MaxListAssets:    DefaultMaxListAssets,
	}
}

func ExamplePaths(root string) (string, string, string) {
	clean := filepath.Clean(root)
	return clean, filepath.Join(clean, "content"), filepath.Join(clean, "static")
}
