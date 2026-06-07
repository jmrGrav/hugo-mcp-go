package hooks

import (
	"context"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
)

type HookEvent struct {
	Mutation string
	Action   string
	URLs     []string
}

type HookProvider interface {
	Name() string
	Run(context.Context, string, []string) (HookRunResult, error)
}

type Processor interface {
	Process(context.Context, HookEvent) HookSummary
}

type Pipeline struct {
	cfg       Config
	store     *Store
	providers []HookProvider
}

type HookSummary struct {
	HooksEnabled     bool
	CloudflarePurge  HookRunResult
	GoogleIndexing   HookRunResult
	IndexNow         HookRunResult
	QueuedURLsCount   int
	FailedJobsCount   int
}

func NewPipeline(cfg Config, store *Store, providers ...HookProvider) *Pipeline {
	return &Pipeline{cfg: cfg, store: store, providers: providers}
}

func (p *Pipeline) Process(ctx context.Context, event HookEvent) HookSummary {
	urls := dedupeStrings(event.URLs)
	summary := HookSummary{
		HooksEnabled:   p != nil && p.cfg.PostBuildHooksEnabled,
		QueuedURLsCount: len(urls),
	}
	if p == nil {
		return summary
	}
	for _, provider := range p.providers {
		if p.store != nil && len(urls) > 0 {
			_, _ = p.store.Enqueue(ctx, HookJob{
				Provider:   provider.Name(),
				Action:     event.Action,
				TargetURLs: append([]string(nil), urls...),
				Status:     "pending",
			})
		}
		if !p.cfg.PostBuildHooksEnabled {
			continue
		}
		result, err := provider.Run(ctx, event.Action, urls)
		switch provider.Name() {
		case "cloudflare":
			summary.CloudflarePurge = result
		case "google_indexing":
			summary.GoogleIndexing = result
		case "indexnow":
			summary.IndexNow = result
		}
		if err != nil {
			summary.FailedJobsCount++
			if p.store != nil {
				_ = p.store.RecordAudit(ctx, AuditRecord{
					Provider: provider.Name(),
					Action:   event.Action,
					Message:  observability.RedactString(redactSecrets(err.Error())),
				})
			}
		}
	}
	return summary
}

func dedupeStrings(values []string) []string {
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
