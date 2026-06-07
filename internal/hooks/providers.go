package hooks

import (
	"strings"
)

type HookRunResult struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	URLCount int    `json:"url_count"`
	Attempts int    `json:"attempts"`
	DryRun   bool   `json:"dry_run"`
	Message  string `json:"message,omitempty"`
}

func redactSecrets(s string, secrets ...string) string {
	out := s
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		out = strings.ReplaceAll(out, secret, "<redacted>")
	}
	return out
}
