package tools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/assets"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/pages"
	"github.com/jmrGrav/hugo-mcp-go/internal/sri"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Deps struct {
	Pages             *pages.Service
	Assets            *assets.Service
	PageMutations     *mutations.PageService
	AssetMutations    *mutations.AssetService
	Build             *mutations.BuildService
	Sri               *sri.Service
	Hooks             hooks.Processor
	HooksStore        *hooks.Store
	HooksAdminEnabled bool
	SiteBaseURL       string
}

type listPagesInput struct {
	Lang    string `json:"lang,omitempty"`
	Section string `json:"section,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type listPagesOutput = pages.ListResult

type getPageInput struct {
	Route string `json:"route"`
	Lang  string `json:"lang,omitempty"`
}

type getPageOutput = pages.Page

type getPageChunkInput struct {
	Route      string `json:"route"`
	Lang       string `json:"lang,omitempty"`
	Cursor     int    `json:"cursor,omitempty"`
	ChunkBytes int    `json:"chunk_bytes,omitempty"`
}

type getPageChunkOutput = pages.ChunkResult

type listAssetsInput struct {
	Type       string `json:"type,omitempty" jsonschema:"asset type filter: image, document, or all"`
	PathPrefix string `json:"path_prefix,omitempty" jsonschema:"relative prefix under the Hugo content/ or static/ roots; use an exact directory prefix such as posts/bonjour"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"maximum number of assets to return"`
	Cursor     string `json:"cursor,omitempty" jsonschema:"pagination cursor from a previous list_assets call"`
}

type listAssetsOutput = assets.ListResult

type getAssetChunkInput struct {
	Path       string `json:"path"`
	Cursor     int    `json:"cursor,omitempty"`
	ChunkBytes int    `json:"chunk_bytes,omitempty"`
}

type getAssetChunkOutput = assets.ChunkResult

type createPageInput struct {
	Route       string   `json:"route"`
	Lang        string   `json:"lang,omitempty"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
	Draft       *bool    `json:"draft,omitempty"`
	Frontmatter any      `json:"frontmatter,omitempty"`
}

type updatePageInput struct {
	Route       string    `json:"route"`
	Lang        string    `json:"lang,omitempty"`
	Title       *string   `json:"title,omitempty"`
	Content     *string   `json:"content,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	Draft       *bool     `json:"draft,omitempty"`
	Frontmatter any       `json:"frontmatter,omitempty"`
}

type deletePageInput struct {
	Route string `json:"route"`
	Lang  string `json:"lang,omitempty"`
}

type uploadAssetInput struct {
	Filename  string `json:"filename"`
	Data      string `json:"data"`
	Subfolder string `json:"subfolder,omitempty"`
}

type buildSiteInput struct {
	PurgeCF bool `json:"purge_cf,omitempty"`
}

func Register(s *mcp.Server, deps Deps) {
	mcp.AddTool(s, toolMeta("list_pages", "List pages", "List Hugo pages.", true), func(ctx context.Context, _ *mcp.CallToolRequest, in listPagesInput) (*mcp.CallToolResult, listPagesOutput, error) {
		if deps.Pages == nil {
			return nil, listPagesOutput{}, errors.New("pages service not configured")
		}
		res, err := deps.Pages.List(ctx, pages.ListRequest{Lang: in.Lang, Section: in.Section, Cursor: in.Cursor, Limit: in.Limit})
		if err != nil {
			return nil, listPagesOutput{}, err
		}
		return nil, listPagesOutput(res), nil
	})

	mcp.AddTool(s, toolMeta("get_page", "Get page", "Get a single Hugo page.", true), func(ctx context.Context, _ *mcp.CallToolRequest, in getPageInput) (*mcp.CallToolResult, getPageOutput, error) {
		if deps.Pages == nil {
			return nil, getPageOutput{}, errors.New("pages service not configured")
		}
		res, err := deps.Pages.Get(ctx, pages.GetRequest{Route: in.Route, Lang: in.Lang})
		if err != nil {
			return nil, getPageOutput{}, err
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("get_page_chunk", "Get page chunk", "Get a chunk of a large Hugo page.", true), func(ctx context.Context, _ *mcp.CallToolRequest, in getPageChunkInput) (*mcp.CallToolResult, getPageChunkOutput, error) {
		if deps.Pages == nil {
			return nil, getPageChunkOutput{}, errors.New("pages service not configured")
		}
		res, err := deps.Pages.GetChunk(ctx, pages.ChunkRequest{Route: in.Route, Lang: in.Lang, Cursor: in.Cursor, ChunkBytes: in.ChunkBytes})
		if err != nil {
			return nil, getPageChunkOutput{}, err
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("create_page", "Create page", "Create a new Hugo page.", false), func(ctx context.Context, _ *mcp.CallToolRequest, in createPageInput) (*mcp.CallToolResult, mutations.MutationResult, error) {
		if deps.PageMutations == nil {
			return nil, mutations.MutationResult{}, errors.New("page mutation service not configured")
		}
		res, err := deps.PageMutations.Create(ctx, mutations.CreatePageRequest{
			Route:       in.Route,
			Lang:        in.Lang,
			Title:       in.Title,
			Content:     in.Content,
			Tags:        in.Tags,
			Draft:       in.Draft,
			Frontmatter: in.Frontmatter,
		})
		if err != nil {
			return nil, mutations.MutationResult{}, err
		}
		if summary, ok := buildHookSummary(ctx, deps, "create_page", "URL_UPDATED", siteURLForRoute(deps.SiteBaseURL, in.Route)); ok {
			applyMutationHookSummary(&res, summary)
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("update_page", "Update page", "Update an existing Hugo page.", false), func(ctx context.Context, _ *mcp.CallToolRequest, in updatePageInput) (*mcp.CallToolResult, mutations.MutationResult, error) {
		if deps.PageMutations == nil {
			return nil, mutations.MutationResult{}, errors.New("page mutation service not configured")
		}
		res, err := deps.PageMutations.Update(ctx, mutations.UpdatePageRequest{
			Route:       in.Route,
			Lang:        in.Lang,
			Title:       in.Title,
			Content:     in.Content,
			Tags:        in.Tags,
			Draft:       in.Draft,
			Frontmatter: in.Frontmatter,
		})
		if err != nil {
			return nil, mutations.MutationResult{}, err
		}
		if summary, ok := buildHookSummary(ctx, deps, "update_page", "URL_UPDATED", siteURLForRoute(deps.SiteBaseURL, in.Route)); ok {
			applyMutationHookSummary(&res, summary)
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("delete_page", "Delete page", "Delete a Hugo page.", false), func(ctx context.Context, _ *mcp.CallToolRequest, in deletePageInput) (*mcp.CallToolResult, mutations.MutationResult, error) {
		if deps.PageMutations == nil {
			return nil, mutations.MutationResult{}, errors.New("page mutation service not configured")
		}
		res, err := deps.PageMutations.Delete(ctx, mutations.DeletePageRequest{
			Route: in.Route,
			Lang:  in.Lang,
		})
		if err != nil {
			return nil, mutations.MutationResult{}, err
		}
		if summary, ok := buildHookSummary(ctx, deps, "delete_page", "URL_DELETED", siteURLForRoute(deps.SiteBaseURL, in.Route)); ok {
			applyMutationHookSummary(&res, summary)
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("build_site", "Build site", "Build the Hugo site.", false), func(ctx context.Context, req *mcp.CallToolRequest, in buildSiteInput) (*mcp.CallToolResult, mutations.BuildResult, error) {
		if deps.Build == nil {
			return nil, mutations.BuildResult{}, errors.New("build service not configured")
		}
		emitProgress(ctx, req, "build_site started", 0, 3)
		res, err := deps.Build.Build(ctx, mutations.BuildRequest{PurgeCF: in.PurgeCF})
		if err != nil {
			emitProgress(ctx, req, "build_site failed", 0, 3)
			return nil, mutations.BuildResult{}, err
		}
		emitProgress(ctx, req, "build_site completed", 3, 3)
		if summary, ok := buildHookSummary(ctx, deps, "build_site", "URL_UPDATED", siteURLForRoute(deps.SiteBaseURL, "/")); ok {
			applyBuildHookSummary(&res, summary)
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("upload_asset", "Upload asset", "Upload a static asset.", false), func(ctx context.Context, _ *mcp.CallToolRequest, in uploadAssetInput) (*mcp.CallToolResult, mutations.UploadAssetResult, error) {
		if deps.AssetMutations == nil {
			return nil, mutations.UploadAssetResult{}, errors.New("asset mutation service not configured")
		}
		res, err := deps.AssetMutations.UploadAsset(ctx, mutations.UploadAssetRequest{
			Filename:  in.Filename,
			Data:      in.Data,
			Subfolder: in.Subfolder,
		})
		if err != nil {
			return nil, mutations.UploadAssetResult{}, err
		}
		if summary, ok := buildHookSummary(ctx, deps, "upload_asset", "URL_UPDATED", siteURLForPath(deps.SiteBaseURL, res.PublicURL)); ok {
			applyUploadHookSummary(&res, summary)
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("list_assets", "List assets", "List Hugo assets.", true), func(ctx context.Context, _ *mcp.CallToolRequest, in listAssetsInput) (*mcp.CallToolResult, listAssetsOutput, error) {
		if deps.Assets == nil {
			return nil, listAssetsOutput{}, errors.New("assets service not configured")
		}
		res, err := deps.Assets.List(ctx, assets.ListRequest{Type: in.Type, PathPrefix: in.PathPrefix, MaxResults: in.MaxResults, Cursor: in.Cursor})
		if err != nil {
			return nil, listAssetsOutput{}, err
		}
		// Keep raw modified values in the structured output for traceability,
		// but committed parity tests compare a normalized JSON fixture that
		// strips them.
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("get_asset_chunk", "Get asset chunk", "Get a chunk of a large Hugo asset.", true), func(ctx context.Context, _ *mcp.CallToolRequest, in getAssetChunkInput) (*mcp.CallToolResult, getAssetChunkOutput, error) {
		if deps.Assets == nil {
			return nil, getAssetChunkOutput{}, errors.New("assets service not configured")
		}
		res, err := deps.Assets.GetChunk(ctx, assets.ChunkRequest{Path: in.Path, Cursor: in.Cursor, ChunkBytes: in.ChunkBytes})
		if err != nil {
			return nil, getAssetChunkOutput{}, err
		}
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("check_sri_versions", "Check SRI versions", "Audit SRI hashes and npm versions for CDN libraries.", false), func(ctx context.Context, req *mcp.CallToolRequest, in checkSriVersionsInput) (*mcp.CallToolResult, checkSriVersionsOutput, error) {
		emitProgress(ctx, req, "check_sri_versions started", 0, 2)
		if deps.Sri == nil {
			emitProgress(ctx, req, "check_sri_versions warning: SRI audit disabled", 1, 2)
		}
		res, err := checkSriVersions(ctx, deps, in)
		if err != nil {
			emitProgress(ctx, req, "check_sri_versions failed", 0, 2)
			return nil, checkSriVersionsOutput{}, err
		}
		emitProgress(ctx, req, "check_sri_versions completed", 2, 2)
		return nil, res, nil
	})

	mcp.AddTool(s, toolMeta("generate_featured_image", "Generate featured image", "Generate a featured image and optionally update page frontmatter.", false), func(ctx context.Context, req *mcp.CallToolRequest, in generateFeaturedImageInput) (*mcp.CallToolResult, generateFeaturedImageOutput, error) {
		emitProgress(ctx, req, "generate_featured_image started", 0, 2)
		res, err := generateFeaturedImage(ctx, deps, in)
		if err != nil {
			emitProgress(ctx, req, "generate_featured_image failed", 0, 2)
			return nil, generateFeaturedImageOutput{}, err
		}
		emitProgress(ctx, req, "generate_featured_image completed", 2, 2)
		return nil, res, nil
	})

	registerHookAdminTools(s, deps)
}

func MustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func toolMeta(name, title, description string, readOnly bool) *mcp.Tool {
	t := &mcp.Tool{Name: name, Title: title, Description: description}
	if readOnly {
		t.Annotations = &mcp.ToolAnnotations{ReadOnlyHint: true}
		return t
	}
	t.Annotations = &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)}
	return t
}

func boolPtr(v bool) *bool {
	return &v
}
