package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
)

type CloudflareClient struct {
	cfg        Config
	httpClient *http.Client
	baseURL    string
}

func NewCloudflareClient(cfg Config, httpClient *http.Client) *CloudflareClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &CloudflareClient{
		cfg:        cfg,
		httpClient: httpClient,
		baseURL:    "https://api.cloudflare.com/client/v4",
	}
}

func (c *CloudflareClient) Name() string { return "cloudflare" }

func (c *CloudflareClient) Run(ctx context.Context, action string, urls []string) (HookRunResult, error) {
	_ = action
	return c.PurgeURLs(ctx, urls)
}

func (c *CloudflareClient) PurgeURLs(ctx context.Context, urls []string) (HookRunResult, error) {
	if err := ctx.Err(); err != nil {
		return HookRunResult{}, err
	}
	if len(urls) == 0 {
		return HookRunResult{}, errors.New("missing target urls")
	}
	if err := validateHTTPSURLs(urls); err != nil {
		return HookRunResult{}, err
	}
	if c == nil {
		return HookRunResult{}, errors.New("cloudflare client not configured")
	}
	if !c.cfg.CloudflarePurgeEnabled || c.cfg.HooksDryRun {
		return HookRunResult{
			Provider: "cloudflare",
			Status:   "dry_run",
			URLCount: len(urls),
			DryRun:   true,
		}, nil
	}
	token, err := LoadSecretFile(c.cfg.CloudflareTokenFile, filepath.Dir(c.cfg.CloudflareTokenFile))
	if err != nil {
		return HookRunResult{}, err
	}
	reqBody, err := json.Marshal(map[string]any{"files": urls})
	if err != nil {
		return HookRunResult{}, err
	}
	endpoint := fmt.Sprintf("%s/zones/%s/purge_cache", strings.TrimRight(c.baseURL, "/"), url.PathEscape(c.cfg.CloudflareZoneID))
	attempts := 0
	var lastErr error
	for attempts < maxRetries(c.cfg.HooksMaxRetries) {
		attempts++
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return HookRunResult{}, err
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("cloudflare purge request failed: %w", err)
			continue
		}
		body, _ := ioReadAllLimit(resp.Body, 64<<10)
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return HookRunResult{
				Provider: "cloudflare",
				Status:   "ok",
				URLCount: len(urls),
				Attempts: attempts,
			}, nil
		}
		lastErr = fmt.Errorf("cloudflare purge failed: %s", strings.TrimSpace(string(body)))
	}
	if lastErr == nil {
		lastErr = errors.New("cloudflare purge failed")
	}
	return HookRunResult{}, errors.New(observability.RedactString(redactSecrets(lastErr.Error(), string(token))))
}

func validateHTTPSURLs(urls []string) error {
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return errors.New("invalid target url: empty")
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "https" && parsed.Scheme != "http" || parsed.Host == "" {
			return fmt.Errorf("invalid target url: %s", raw)
		}
	}
	return nil
}

func maxRetries(v int) int {
	if v <= 0 {
		return DefaultHooksMaxRetries
	}
	return v
}
