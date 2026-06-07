package mutations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
	"github.com/jmrGrav/hugo-mcp-go/internal/runner"
)

type BuildRequest struct {
	PurgeCF bool
}

type BuildResult struct {
	Status          string         `json:"status"`
	Deploy          string         `json:"deploy"`
	CFPurge         map[string]any `json:"cf_purge"`
	HooksEnabled    bool           `json:"hooks.enabled,omitempty"`
	CloudflarePurge map[string]any `json:"cloudflare_purge,omitempty"`
	GoogleIndexing  map[string]any `json:"google_indexing,omitempty"`
	IndexNow        map[string]any `json:"indexnow,omitempty"`
	QueuedURLsCount int             `json:"queued_urls_count,omitempty"`
	FailedJobsCount int              `json:"failed_jobs_count,omitempty"`
}

type BuildService struct {
	Stage   *staging.Workspace
	Runner  runner.Runner
	Timeout time.Duration
}

func NewBuildService(ws *staging.Workspace, r runner.Runner) *BuildService {
	return &BuildService{
		Stage:   ws,
		Runner:  r,
		Timeout: 5 * time.Minute,
	}
}

func (s *BuildService) Build(ctx context.Context, req BuildRequest) (BuildResult, error) {
	if ctx.Err() != nil {
		return BuildResult{}, ctx.Err()
	}
	if s == nil || s.Stage == nil {
		return BuildResult{}, errors.New("missing staging workspace")
	}
	if s.Runner == nil {
		return BuildResult{}, errors.New("missing runner")
	}
	buildCtx := ctx
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	buildCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, stderr, err := s.Runner.Run(buildCtx, "hugo",
		"--source", s.Stage.HugoRoot,
		"--destination", s.Stage.PublicRoot,
	)
	if err != nil {
		stderr = strings.TrimSpace(stderr)
		if stderr != "" {
			return BuildResult{}, fmt.Errorf("build_site failed: %w: %s", err, stderr)
		}
		return BuildResult{}, fmt.Errorf("build_site failed: %w", err)
	}
	return BuildResult{
		Status:  "built",
		Deploy:  "DEPLOY_SKIPPED",
		CFPurge: map[string]any{"skipped": "use cloudflare plugin"},
	}, nil
}
