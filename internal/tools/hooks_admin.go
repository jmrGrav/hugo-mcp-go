package tools

import (
	"context"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type hookSummaryOutput struct {
	HooksEnabled    bool           `json:"hooks.enabled"`
	CloudflarePurge map[string]any `json:"cloudflare_purge"`
	GoogleIndexing  map[string]any `json:"google_indexing"`
	IndexNow        map[string]any `json:"indexnow"`
	QueuedURLsCount int            `json:"queued_urls_count"`
	FailedJobsCount int            `json:"failed_jobs_count"`
}

type listHookJobsOutput struct {
	Total int             `json:"total"`
	Jobs  []hookJobOutput `json:"jobs"`
}

type hookJobOutput struct {
	ID         string   `json:"id"`
	Provider   string   `json:"provider"`
	Action     string   `json:"action"`
	TargetURLs []string `json:"target_urls"`
	Status     string   `json:"status"`
	LastError  string   `json:"last_error,omitempty"`
}

type retryHookJobsInput struct {
	JobIDs []string `json:"job_ids,omitempty"`
}

type retryHookJobsOutput struct {
	RetriedCount int `json:"retried_count"`
}

type runPostBuildHooksInput struct {
	Action string   `json:"action"`
	URLs   []string `json:"urls"`
}

func emptyHookSummaryOutput() hookSummaryOutput {
	return hookSummaryOutput{
		CloudflarePurge: map[string]any{},
		GoogleIndexing:  map[string]any{},
		IndexNow:        map[string]any{},
	}
}

func registerHookAdminTools(s *mcp.Server, deps Deps) {
	if !deps.HooksAdminEnabled {
		return
	}

	mcp.AddTool(s, &mcp.Tool{Name: "list_hook_jobs", Description: "List pending and failed hook jobs."}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listHookJobsOutput, error) {
		if deps.HooksStore == nil {
			return nil, listHookJobsOutput{}, context.Canceled
		}
		jobs, err := deps.HooksStore.ListJobs(ctx)
		if err != nil {
			return nil, listHookJobsOutput{}, err
		}
		out := listHookJobsOutput{Total: len(jobs), Jobs: make([]hookJobOutput, 0, len(jobs))}
		for _, job := range jobs {
			out.Jobs = append(out.Jobs, hookJobOutput{
				ID:         job.ID,
				Provider:   job.Provider,
				Action:     job.Action,
				TargetURLs: append([]string(nil), job.TargetURLs...),
				Status:     job.Status,
				LastError:  observability.RedactString(job.LastError),
			})
		}
		return nil, out, nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "retry_hook_jobs", Description: "Retry failed hook jobs."}, func(ctx context.Context, _ *mcp.CallToolRequest, in retryHookJobsInput) (*mcp.CallToolResult, retryHookJobsOutput, error) {
		if deps.HooksStore == nil {
			return nil, retryHookJobsOutput{}, context.Canceled
		}
		jobIDs := append([]string(nil), in.JobIDs...)
		if len(jobIDs) == 0 {
			jobs, err := deps.HooksStore.ListJobs(ctx)
			if err != nil {
				return nil, retryHookJobsOutput{}, err
			}
			for _, job := range jobs {
				if job.Status == "failed" {
					jobIDs = append(jobIDs, job.ID)
				}
			}
		}
		jobs, err := deps.HooksStore.ListJobs(ctx)
		if err != nil {
			return nil, retryHookJobsOutput{}, err
		}
		byID := make(map[string]hooks.HookJob, len(jobs))
		for _, job := range jobs {
			byID[job.ID] = job
		}
		var retried int
		for _, id := range jobIDs {
			job, ok := byID[id]
			if !ok {
				continue
			}
			retried++
			_, _ = deps.HooksStore.SetJobStatus(ctx, []string{id}, "pending")
			if deps.Hooks != nil {
				_ = deps.Hooks.Process(ctx, hooks.HookEvent{
					Mutation: "retry_hook_jobs",
					Action:   job.Action,
					URLs:     append([]string(nil), job.TargetURLs...),
				})
			}
		}
		return nil, retryHookJobsOutput{RetriedCount: retried}, nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "get_hook_status", Description: "Get hooks summary and queue counts."}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, hookSummaryOutput, error) {
		if deps.HooksStore == nil {
			return nil, emptyHookSummaryOutput(), context.Canceled
		}
		jobs, err := deps.HooksStore.ListJobs(ctx)
		if err != nil {
			return nil, emptyHookSummaryOutput(), err
		}
		out := emptyHookSummaryOutput()
		out.HooksEnabled = deps.Hooks != nil
		queuedURLs := map[string]struct{}{}
		for _, job := range jobs {
			for _, u := range job.TargetURLs {
				queuedURLs[u] = struct{}{}
			}
			switch job.Status {
			case "failed":
				out.FailedJobsCount++
			}
		}
		out.QueuedURLsCount = len(queuedURLs)
		return nil, out, nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "run_post_build_hooks", Description: "Run post-build hooks for explicit URLs."}, func(ctx context.Context, _ *mcp.CallToolRequest, in runPostBuildHooksInput) (*mcp.CallToolResult, hookSummaryOutput, error) {
		if deps.Hooks == nil {
			return nil, emptyHookSummaryOutput(), context.Canceled
		}
		summary := deps.Hooks.Process(ctx, hooks.HookEvent{
			Mutation: "run_post_build_hooks",
			Action:   in.Action,
			URLs:     append([]string(nil), in.URLs...),
		})
		out := emptyHookSummaryOutput()
		out.HooksEnabled = summary.HooksEnabled
		out.CloudflarePurge = summary.CloudflarePurge.MCP()
		out.GoogleIndexing = summary.GoogleIndexing.MCP()
		out.IndexNow = summary.IndexNow.MCP()
		out.QueuedURLsCount = summary.QueuedURLsCount
		out.FailedJobsCount = summary.FailedJobsCount
		return nil, out, nil
	})
}
