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

type IndexNowClient struct {
	cfg        Config
	httpClient *http.Client
}

func NewIndexNowClient(cfg Config, httpClient *http.Client) *IndexNowClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &IndexNowClient{cfg: cfg, httpClient: httpClient}
}

func (c *IndexNowClient) Name() string { return "indexnow" }

func (c *IndexNowClient) Run(ctx context.Context, action string, urls []string) (HookRunResult, error) {
	return c.Submit(ctx, action, urls)
}

func (c *IndexNowClient) Submit(ctx context.Context, action string, urls []string) (HookRunResult, error) {
	if err := ctx.Err(); err != nil {
		return HookRunResult{}, err
	}
	if len(urls) == 0 {
		return HookRunResult{}, errors.New("missing target urls")
	}
	if err := validateHTTPSURLs(urls); err != nil {
		return HookRunResult{}, err
	}
	action = strings.ToUpper(strings.TrimSpace(action))
	if action != "URL_UPDATED" && action != "URL_DELETED" {
		return HookRunResult{}, fmt.Errorf("invalid IndexNow action %q", action)
	}
	if c == nil {
		return HookRunResult{}, errors.New("indexnow client not configured")
	}
	if !c.cfg.IndexNowEnabled || c.cfg.HooksDryRun {
		return HookRunResult{
			Provider: "indexnow",
			Status:   "dry_run",
			URLCount: len(urls),
			DryRun:   true,
		}, nil
	}
	keyRaw, err := LoadSecretFile(c.cfg.IndexNowKeyFile, filepath.Dir(c.cfg.IndexNowKeyFile))
	if err != nil {
		return HookRunResult{}, err
	}
	key := strings.TrimSpace(string(keyRaw))
	first, err := url.Parse(urls[0])
	if err != nil {
		return HookRunResult{}, err
	}
	host := first.Host
	if host == "" {
		return HookRunResult{}, errors.New("missing host in target url")
	}
	payload := map[string]any{
		"host":        host,
		"key":         key,
		"keyLocation":  fmt.Sprintf("https://%s/%s.txt", host, url.PathEscape(key)),
		"urlList":     urls,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return HookRunResult{}, err
	}
	attempts := 0
	var lastErr error
	for attempts < maxRetries(c.cfg.HooksMaxRetries) {
		attempts++
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.IndexNowEndpoint, bytes.NewReader(body))
		if err != nil {
			return HookRunResult{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("indexnow request failed: %w", err)
			continue
		}
		respBody, _ := ioReadAllLimit(resp.Body, 64<<10)
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return HookRunResult{
				Provider: "indexnow",
				Status:   "ok",
				URLCount: len(urls),
				Attempts: attempts,
			}, nil
		}
		lastErr = fmt.Errorf("indexnow failed: %s", strings.TrimSpace(string(respBody)))
	}
	if lastErr == nil {
		lastErr = errors.New("indexnow failed")
	}
	return HookRunResult{}, errors.New(observability.RedactString(redactSecrets(lastErr.Error(), key)))
}
