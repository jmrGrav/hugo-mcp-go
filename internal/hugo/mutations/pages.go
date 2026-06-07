package mutations

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/frontmatter"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/pages"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

type PageService struct {
	Stage        *staging.Workspace
	Reader       *pages.Service
	Now          func() time.Time
	MaxPageBytes int64
}

type CreatePageRequest struct {
	Route       string
	Lang        string
	Title       string
	Content     string
	Tags        []string
	Draft       *bool
	Frontmatter any
}

type UpdatePageRequest struct {
	Route       string
	Lang        string
	Title       *string
	Content     *string
	Tags        *[]string
	Draft       *bool
	Frontmatter any
}

type DeletePageRequest struct {
	Route string
	Lang  string
}

type MutationResult struct {
	Status         string         `json:"status"`
	File           string         `json:"file"`
	Deploy         string         `json:"deploy"`
	CFPurge        map[string]any `json:"cf_purge"`
	Plugins        []any          `json:"plugins"`
	HooksEnabled   bool           `json:"hooks.enabled,omitempty"`
	CloudflarePurge map[string]any `json:"cloudflare_purge,omitempty"`
	GoogleIndexing map[string]any `json:"google_indexing,omitempty"`
	IndexNow       map[string]any `json:"indexnow,omitempty"`
	QueuedURLsCount int            `json:"queued_urls_count,omitempty"`
	FailedJobsCount int             `json:"failed_jobs_count,omitempty"`
}

func NewPageService(ws *staging.Workspace) *PageService {
	return &PageService{
		Stage:        ws,
		Reader:       pages.New(ws.ContentRoot),
		Now:          time.Now,
		MaxPageBytes: 1 << 20,
	}
}

func (s *PageService) Create(ctx context.Context, req CreatePageRequest) (MutationResult, error) {
	if err := ctx.Err(); err != nil {
		return MutationResult{}, err
	}
	if s == nil || s.Stage == nil || s.Reader == nil {
		return MutationResult{}, fmt.Errorf("missing staging workspace")
	}
	if strings.TrimSpace(req.Route) == "" {
		return MutationResult{}, fmt.Errorf("missing required field: route")
	}
	if strings.TrimSpace(req.Title) == "" {
		return MutationResult{}, fmt.Errorf("missing required field: title")
	}
	if strings.TrimSpace(req.Content) == "" {
		return MutationResult{}, fmt.Errorf("missing required field: content")
	}
	route, root, err := normalizeMutationRoute(req.Route)
	if err != nil {
		return MutationResult{}, err
	}
	lang, err := normalizeCreateLang(req.Lang)
	if err != nil {
		return MutationResult{}, err
	}
	dedicated := map[string]any{"title": req.Title, "tags": req.Tags, "draft": req.Draft}
	fm, err := validateFrontmatter(req.Frontmatter, false, dedicated)
	if err != nil {
		return MutationResult{}, err
	}
	if _, ok := fm["date"]; !ok {
		fm["date"] = s.nowString()
	}
	if _, ok := fm["lastmod"]; !ok {
		fm["lastmod"] = s.nowString()
	}
	final := map[string]any{"title": req.Title}
	if req.Tags != nil {
		final["tags"] = req.Tags
	}
	if req.Draft != nil {
		final["draft"] = *req.Draft
	}
	final = deepMerge(final, fm)
	rel := pageRelativePath(root, route, lang)
	if err := writeAtomicAt(s.Stage.ContentRoot, rel, final, req.Content, true, s.maxPageBytes()); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{
		Status:  "created",
		File:    filepath.ToSlash(rel),
		Deploy:  "DEPLOY_SKIPPED",
		CFPurge: map[string]any{"skipped": "plugin not active"},
		Plugins: []any{},
	}, nil
}

func (s *PageService) Update(ctx context.Context, req UpdatePageRequest) (MutationResult, error) {
	if err := ctx.Err(); err != nil {
		return MutationResult{}, err
	}
	if s == nil || s.Stage == nil || s.Reader == nil {
		return MutationResult{}, fmt.Errorf("missing staging workspace")
	}
	if strings.TrimSpace(req.Route) == "" {
		return MutationResult{}, fmt.Errorf("missing required field: route")
	}
	route, root, err := normalizeMutationRoute(req.Route)
	if err != nil {
		return MutationResult{}, err
	}
	lang, err := normalizeOptionalLang(req.Lang)
	if err != nil {
		return MutationResult{}, err
	}
	existing, err := s.Reader.Get(ctx, pages.GetRequest{Route: mutationGetRoute(root, route), Lang: lang})
	if err != nil {
		if strings.HasPrefix(err.Error(), "Page not found: ") {
			return MutationResult{}, fmt.Errorf("Page not found: %s", route)
		}
		return MutationResult{}, err
	}
	dedicated := map[string]any{"title": req.Title, "tags": req.Tags, "draft": req.Draft}
	fm, err := validateFrontmatter(req.Frontmatter, true, dedicated)
	if err != nil {
		return MutationResult{}, err
	}
	final := frontmatter.CloneMap(existing.Frontmatter)
	if req.Title != nil {
		final["title"] = *req.Title
	}
	if req.Tags != nil {
		final["tags"] = *req.Tags
	}
	if req.Draft != nil {
		final["draft"] = *req.Draft
	}
	if _, ok := fm["lastmod"]; !ok {
		fm["lastmod"] = s.nowString()
	}
	final = deepMerge(final, fm)
	body := existing.Content
	if req.Content != nil {
		body = *req.Content
	}
	if _, err := pathguard.OpenExistingFile(s.Stage.ContentRoot, existing.File); err != nil {
		return MutationResult{}, err
	}
	if err := writeAtomicAt(s.Stage.ContentRoot, existing.File, final, body, false, s.maxPageBytes()); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{
		Status:  "updated",
		File:    existing.File,
		Deploy:  "DEPLOY_SKIPPED",
		CFPurge: map[string]any{"skipped": "plugin not active"},
		Plugins: []any{},
	}, nil
}

func (s *PageService) Delete(ctx context.Context, req DeletePageRequest) (MutationResult, error) {
	if err := ctx.Err(); err != nil {
		return MutationResult{}, err
	}
	if s == nil || s.Stage == nil || s.Reader == nil {
		return MutationResult{}, fmt.Errorf("missing staging workspace")
	}
	if strings.TrimSpace(req.Route) == "" {
		return MutationResult{}, fmt.Errorf("missing required field: route")
	}
	route, root, err := normalizeMutationRoute(req.Route)
	if err != nil {
		return MutationResult{}, err
	}
	lang, err := normalizeOptionalLang(req.Lang)
	if err != nil {
		return MutationResult{}, err
	}
	existing, err := s.Reader.Get(ctx, pages.GetRequest{Route: mutationGetRoute(root, route), Lang: lang})
	if err != nil {
		if strings.HasPrefix(err.Error(), "Page not found: ") {
			return MutationResult{}, fmt.Errorf("Page not found: %s", route)
		}
		return MutationResult{}, err
	}
	workDir, err := pathguard.OpenDirChain(s.Stage.WorkRoot, ".", true)
	if err != nil {
		return MutationResult{}, err
	}
	defer workDir.Close()
	backupName, backupFile, err := pathguard.CreateTempFile(workDir, "delete-", 0o644)
	if err != nil {
		return MutationResult{}, err
	}
	file, err := pathguard.OpenExistingFile(s.Stage.ContentRoot, existing.File)
	if err != nil {
		_ = backupFile.Close()
		_ = pathguard.UnlinkInDir(workDir, backupName)
		return MutationResult{}, err
	}
	raw, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		_ = backupFile.Close()
		_ = pathguard.UnlinkInDir(workDir, backupName)
		return MutationResult{}, err
	}
	if _, err := backupFile.Write(raw); err != nil {
		_ = backupFile.Close()
		_ = pathguard.UnlinkInDir(workDir, backupName)
		return MutationResult{}, err
	}
	if err := backupFile.Close(); err != nil {
		_ = pathguard.UnlinkInDir(workDir, backupName)
		return MutationResult{}, err
	}
	parentFD, err := pathguard.OpenDirChain(s.Stage.ContentRoot, filepath.Dir(existing.File), false)
	if err != nil {
		_ = pathguard.UnlinkInDir(workDir, backupName)
		return MutationResult{}, err
	}
	defer parentFD.Close()
	if err := pathguard.UnlinkInDir(parentFD, filepath.Base(existing.File)); err != nil {
		_ = pathguard.UnlinkInDir(workDir, backupName)
		return MutationResult{}, err
	}
	return MutationResult{
		Status:  "deleted",
		File:    existing.File,
		Deploy:  "DEPLOY_SKIPPED",
		CFPurge: map[string]any{"skipped": "plugin not active"},
		Plugins: []any{},
	}, nil
}

func writeAtomicAt(root, rel string, fm map[string]any, body string, noReplace bool, maxBytes int64) error {
	raw, err := frontmatter.Render(fm, body)
	if err != nil {
		return err
	}
	if limit := maxBytes; limit > 0 && int64(len(raw)) > limit {
		return fmt.Errorf("page too large: %d bytes (max %d)", len(raw), limit)
	}
	return writeBytesAt(root, rel, raw, noReplace)
}

func writeBytesAt(root, rel string, raw []byte, noReplace bool) error {
	dir, err := pathguard.OpenDirChain(root, filepath.Dir(rel), true)
	if err != nil {
		return err
	}
	defer dir.Close()
	tmpName, tmp, err := pathguard.CreateTempFile(dir, ".tmp-", 0o644)
	if err != nil {
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = pathguard.UnlinkInDir(dir, tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = pathguard.UnlinkInDir(dir, tmpName)
		return err
	}
	if err := pathguard.RenameInDir(dir, tmpName, filepath.Base(rel), noReplace); err != nil {
		_ = pathguard.UnlinkInDir(dir, tmpName)
		return err
	}
	return nil
}

func normalizeMutationRoute(route string) (string, bool, error) {
	s := strings.TrimSpace(route)
	s = strings.TrimPrefix(s, "/")
	switch s {
	case "", "_index", "index":
		return "_index", true, nil
	}
	if strings.Contains(s, "\\") || strings.HasPrefix(s, "/") || strings.Contains(s, "..") {
		return "", false, fmt.Errorf("Invalid route (path traversal): %s", route)
	}
	clean, err := pathguard.ValidateRelative(s)
	if err != nil {
		return "", false, fmt.Errorf("Invalid route (path traversal): %s", route)
	}
	return clean, false, nil
}

func normalizeCreateLang(lang string) (string, error) {
	if lang == "" {
		return "fr", nil
	}
	return normalizeOptionalLang(lang)
}

func normalizeOptionalLang(lang string) (string, error) {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return "", nil
	}
	if len(lang) < 2 || len(lang) > 3 {
		return "", fmt.Errorf("Invalid lang (must match ^[a-z]{2,3}$): %s", lang)
	}
	for _, r := range lang {
		if r < 'a' || r > 'z' {
			return "", fmt.Errorf("Invalid lang (must match ^[a-z]{2,3}$): %s", lang)
		}
	}
	return lang, nil
}

func pageRelativePath(rootIsIndex bool, route, lang string) string {
	if rootIsIndex {
		return fmt.Sprintf("_index.%s.md", lang)
	}
	return filepath.ToSlash(filepath.Join(route, fmt.Sprintf("index.%s.md", lang)))
}

func mutationGetRoute(rootIsIndex bool, route string) string {
	if rootIsIndex {
		return "_index"
	}
	return "/" + route
}

func (s *PageService) nowString() string {
	if s.Now == nil {
		s.Now = time.Now
	}
	return s.Now().Format("2006-01-02T15:04:05-07:00")
}

func (s *PageService) maxPageBytes() int64 {
	if s == nil || s.MaxPageBytes <= 0 {
		return 1 << 20
	}
	return s.MaxPageBytes
}
