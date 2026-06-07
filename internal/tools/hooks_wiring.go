package tools

import (
	"context"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
)

func buildHookSummary(ctx context.Context, deps Deps, mutation, action, rawURL string) (hooks.HookSummary, bool) {
	if deps.Hooks == nil {
		return hooks.HookSummary{}, false
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return hooks.HookSummary{}, false
	}
	return deps.Hooks.Process(ctx, hooks.HookEvent{
		Mutation: mutation,
		Action:   action,
		URLs:     []string{rawURL},
	}), true
}

func applyMutationHookSummary(result *mutations.MutationResult, summary hooks.HookSummary) {
	if result == nil {
		return
	}
	result.HooksEnabled = summary.HooksEnabled
	result.CloudflarePurge = summary.CloudflarePurge.MCP()
	result.GoogleIndexing = summary.GoogleIndexing.MCP()
	result.IndexNow = summary.IndexNow.MCP()
	result.QueuedURLsCount = summary.QueuedURLsCount
	result.FailedJobsCount = summary.FailedJobsCount
}

func applyBuildHookSummary(result *mutations.BuildResult, summary hooks.HookSummary) {
	if result == nil {
		return
	}
	result.HooksEnabled = summary.HooksEnabled
	result.CloudflarePurge = summary.CloudflarePurge.MCP()
	result.GoogleIndexing = summary.GoogleIndexing.MCP()
	result.IndexNow = summary.IndexNow.MCP()
	result.QueuedURLsCount = summary.QueuedURLsCount
	result.FailedJobsCount = summary.FailedJobsCount
}

func applyUploadHookSummary(result *mutations.UploadAssetResult, summary hooks.HookSummary) {
	if result == nil {
		return
	}
	result.HooksEnabled = summary.HooksEnabled
	result.CloudflarePurge = summary.CloudflarePurge.MCP()
	result.GoogleIndexing = summary.GoogleIndexing.MCP()
	result.IndexNow = summary.IndexNow.MCP()
	result.QueuedURLsCount = summary.QueuedURLsCount
	result.FailedJobsCount = summary.FailedJobsCount
}

func siteURLForRoute(baseURL, route string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	route = strings.TrimSpace(route)
	if baseURL == "" || route == "" {
		return ""
	}
	if route == "/" {
		return baseURL + "/"
	}
	route = "/" + strings.Trim(route, "/")
	if !strings.HasSuffix(route, "/") {
		route += "/"
	}
	return baseURL + route
}

func siteURLForPath(baseURL, rel string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	rel = strings.TrimSpace(rel)
	if baseURL == "" || rel == "" {
		return ""
	}
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	return baseURL + rel
}
