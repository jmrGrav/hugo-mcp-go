package hooks

func (r HookRunResult) MCP() map[string]any {
	return map[string]any{
		"provider":  r.Provider,
		"status":    r.Status,
		"url_count": r.URLCount,
		"attempts":  r.Attempts,
		"dry_run":   r.DryRun,
	}
}

func (s HookSummary) MCP() map[string]any {
	return map[string]any{
		"hooks.enabled":     s.HooksEnabled,
		"cloudflare_purge":  s.CloudflarePurge.MCP(),
		"google_indexing":   s.GoogleIndexing.MCP(),
		"indexnow":          s.IndexNow.MCP(),
		"queued_urls_count":  s.QueuedURLsCount,
		"failed_jobs_count":  s.FailedJobsCount,
	}
}
