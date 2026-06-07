package tools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/assets"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/pages"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Deps struct {
	Pages          *pages.Service
	Assets         *assets.Service
	PageMutations  *mutations.PageService
	AssetMutations *mutations.AssetService
	Build          *mutations.BuildService
	Hooks          hooks.Processor
	HooksStore     *hooks.Store
	HooksAdminEnabled bool
	SiteBaseURL    string
}

type listPagesInput struct {
	Lang    string `json:"lang,omitempty"`
	Section string `json:"section,omitempty"`
}

type listPagesOutput struct {
	Pages   []pages.Summary `json:"pages"`
	Total   int             `json:"total"`
	Skipped *int            `json:"skipped,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type getPageInput struct {
	Route string `json:"route"`
	Lang  string `json:"lang,omitempty"`
}

type getPageOutput = pages.Page

type listAssetsInput struct {
	Type       string `json:"type,omitempty"`
	PathPrefix string `json:"path_prefix,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type listAssetsOutput = assets.ListResult

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
	mcp.AddTool(s, &mcp.Tool{Name: "list_pages", Description: "List Hugo pages."}, func(ctx context.Context, _ *mcp.CallToolRequest, in listPagesInput) (*mcp.CallToolResult, listPagesOutput, error) {
		if deps.Pages == nil {
			return nil, listPagesOutput{}, errors.New("pages service not configured")
		}
		res, err := deps.Pages.List(ctx, pages.ListRequest{Lang: in.Lang, Section: in.Section})
		if err != nil {
			return nil, listPagesOutput{}, err
		}
		return nil, listPagesOutput(res), nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "get_page", Description: "Get a single Hugo page."}, func(ctx context.Context, _ *mcp.CallToolRequest, in getPageInput) (*mcp.CallToolResult, getPageOutput, error) {
		if deps.Pages == nil {
			return nil, getPageOutput{}, errors.New("pages service not configured")
		}
		res, err := deps.Pages.Get(ctx, pages.GetRequest{Route: in.Route, Lang: in.Lang})
		if err != nil {
			return nil, getPageOutput{}, err
		}
		return nil, res, nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "create_page", Description: "Create a new Hugo page."}, func(ctx context.Context, _ *mcp.CallToolRequest, in createPageInput) (*mcp.CallToolResult, mutations.MutationResult, error) {
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

	mcp.AddTool(s, &mcp.Tool{Name: "update_page", Description: "Update an existing Hugo page."}, func(ctx context.Context, _ *mcp.CallToolRequest, in updatePageInput) (*mcp.CallToolResult, mutations.MutationResult, error) {
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

	mcp.AddTool(s, &mcp.Tool{Name: "delete_page", Description: "Delete a Hugo page."}, func(ctx context.Context, _ *mcp.CallToolRequest, in deletePageInput) (*mcp.CallToolResult, mutations.MutationResult, error) {
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

	mcp.AddTool(s, &mcp.Tool{Name: "build_site", Description: "Build the Hugo site."}, func(ctx context.Context, _ *mcp.CallToolRequest, in buildSiteInput) (*mcp.CallToolResult, mutations.BuildResult, error) {
		if deps.Build == nil {
			return nil, mutations.BuildResult{}, errors.New("build service not configured")
		}
		res, err := deps.Build.Build(ctx, mutations.BuildRequest{PurgeCF: in.PurgeCF})
		if err != nil {
			return nil, mutations.BuildResult{}, err
		}
		if summary, ok := buildHookSummary(ctx, deps, "build_site", "URL_UPDATED", siteURLForRoute(deps.SiteBaseURL, "/")); ok {
			applyBuildHookSummary(&res, summary)
		}
		return nil, res, nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "upload_asset", Description: "Upload a static asset."}, func(ctx context.Context, _ *mcp.CallToolRequest, in uploadAssetInput) (*mcp.CallToolResult, mutations.UploadAssetResult, error) {
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

	mcp.AddTool(s, &mcp.Tool{Name: "list_assets", Description: "List Hugo assets."}, func(ctx context.Context, _ *mcp.CallToolRequest, in listAssetsInput) (*mcp.CallToolResult, listAssetsOutput, error) {
		if deps.Assets == nil {
			return nil, listAssetsOutput{}, errors.New("assets service not configured")
		}
		res, err := deps.Assets.List(ctx, assets.ListRequest{Type: in.Type, PathPrefix: in.PathPrefix, MaxResults: in.MaxResults})
		if err != nil {
			return nil, listAssetsOutput{}, err
		}
		// Keep raw modified values in the structured output for traceability,
		// but committed parity tests compare a normalized JSON fixture that
		// strips them.
		return nil, res, nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "check_sri_versions", Description: "Audit SRI hashes and npm versions for CDN libraries."}, func(ctx context.Context, _ *mcp.CallToolRequest, in checkSriVersionsInput) (*mcp.CallToolResult, checkSriVersionsOutput, error) {
		res, err := checkSriVersions(ctx, in)
		if err != nil {
			return nil, checkSriVersionsOutput{}, err
		}
		return nil, res, nil
	})

	mcp.AddTool(s, &mcp.Tool{Name: "generate_featured_image", Description: "Generate a featured image and optionally update page frontmatter."}, func(ctx context.Context, _ *mcp.CallToolRequest, in generateFeaturedImageInput) (*mcp.CallToolResult, generateFeaturedImageOutput, error) {
		res, err := generateFeaturedImage(ctx, deps, in)
		if err != nil {
			return nil, generateFeaturedImageOutput{}, err
		}
		return nil, res, nil
	})

	registerHookAdminTools(s, deps)
}

func MustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
