package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleIndexingClient struct {
	cfg      Config
	httpClient *http.Client
	baseURL  string
	tokenURL string
}

func NewGoogleIndexingClient(cfg Config, httpClient *http.Client) *GoogleIndexingClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GoogleIndexingClient{
		cfg:      cfg,
		httpClient: httpClient,
		baseURL:  "https://indexing.googleapis.com",
		tokenURL: "https://oauth2.googleapis.com/token",
	}
}

func (c *GoogleIndexingClient) Name() string { return "google_indexing" }

func (c *GoogleIndexingClient) Run(ctx context.Context, action string, urls []string) (HookRunResult, error) {
	return c.Publish(ctx, action, urls)
}

func (c *GoogleIndexingClient) Publish(ctx context.Context, action string, urls []string) (HookRunResult, error) {
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
		return HookRunResult{}, fmt.Errorf("invalid Google Indexing action %q", action)
	}
	if c == nil {
		return HookRunResult{}, errors.New("google indexing client not configured")
	}
	if !c.cfg.GoogleIndexingEnabled || c.cfg.HooksDryRun {
		return HookRunResult{
			Provider: "google_indexing",
			Status:   "dry_run",
			URLCount: len(urls),
			DryRun:   true,
		}, nil
	}
	raw, err := LoadSecretFile(c.cfg.GoogleIndexingServiceAccountFile, filepath.Dir(c.cfg.GoogleIndexingServiceAccountFile))
	if err != nil {
		return HookRunResult{}, err
	}
	jwtCfg, err := google.JWTConfigFromJSON(raw, "https://www.googleapis.com/auth/indexing")
	if err != nil {
		return HookRunResult{}, err
	}
	jwtCfg.TokenURL = c.tokenURL
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	ts := jwtCfg.TokenSource(ctx)
	token, err := ts.Token()
	if err != nil {
		return HookRunResult{}, errors.New(observability.RedactString(redactSecrets(err.Error(), string(raw))))
	}
	attempts := 0
	var lastErr error
	for _, target := range urls {
		for attempts < maxRetries(c.cfg.HooksMaxRetries) {
			attempts++
			reqBody, _ := json.Marshal(map[string]string{
				"url":  target,
				"type": action,
			})
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/v3/urlNotifications:publish", bytes.NewReader(reqBody))
			if err != nil {
				return HookRunResult{}, err
			}
			req.Header.Set("Authorization", "Bearer "+token.AccessToken)
			req.Header.Set("Content-Type", "application/json")
			resp, err := c.httpClient.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("google indexing request failed: %w", err)
				if attempts < maxRetries(c.cfg.HooksMaxRetries) {
					continue
				}
				break
			}
			body, _ := ioReadAllLimit(resp.Body, 64<<10)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				lastErr = nil
				break
			}
			lastErr = fmt.Errorf("google indexing failed: %s", strings.TrimSpace(string(body)))
			if attempts < maxRetries(c.cfg.HooksMaxRetries) {
				continue
			}
		}
	}
	if lastErr != nil {
		return HookRunResult{}, errors.New(observability.RedactString(redactSecrets(lastErr.Error(), token.AccessToken)))
	}
	return HookRunResult{
		Provider: "google_indexing",
		Status:   "ok",
		URLCount: len(urls),
		Attempts: attempts,
	}, nil
}
