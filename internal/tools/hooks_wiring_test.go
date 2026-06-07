package tools

import (
	"context"
	"testing"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
)

func TestHookWiringHelperBranches(t *testing.T) {
	if got := siteURLForRoute("", "/posts/x"); got != "" {
		t.Fatalf("expected empty route url with empty base, got %q", got)
	}
	if got := siteURLForRoute("https://example.com", "/"); got != "https://example.com/" {
		t.Fatalf("unexpected root url %q", got)
	}
	if got := siteURLForRoute("https://example.com/", "/posts/x"); got != "https://example.com/posts/x/" {
		t.Fatalf("unexpected route url %q", got)
	}
	if got := siteURLForPath("https://example.com", "images/x.svg"); got != "https://example.com/images/x.svg" {
		t.Fatalf("unexpected asset url %q", got)
	}
	if got := siteURLForPath("", "images/x.svg"); got != "" {
		t.Fatalf("expected empty asset url with empty base, got %q", got)
	}
	var mutationResult *mutations.MutationResult
	applyMutationHookSummary(mutationResult, hooks.HookSummary{})
	var buildResult *mutations.BuildResult
	applyBuildHookSummary(buildResult, hooks.HookSummary{})
	var uploadResult *mutations.UploadAssetResult
	applyUploadHookSummary(uploadResult, hooks.HookSummary{})
	if _, ok := buildHookSummary(context.Background(), Deps{}, "m", "a", ""); ok {
		t.Fatal("expected buildHookSummary to skip empty URL")
	}
	if _, ok := buildHookSummary(context.Background(), Deps{Hooks: &fakeHookPipeline{}}, "m", "a", "https://example.com/x/"); !ok {
		t.Fatal("expected buildHookSummary to pass with URL and hook processor")
	}
}
