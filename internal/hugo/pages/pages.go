package pages

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/frontmatter"
	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

type Service struct {
	ContentRoot  string
	MaxPageBytes int64
}

type ListRequest struct {
	Lang    string
	Section string
}

type Summary struct {
	Route string   `json:"route"`
	Lang  string   `json:"lang"`
	File  string   `json:"file"`
	Title string   `json:"title"`
	Date  string   `json:"date"`
	Draft bool     `json:"draft"`
	Tags  []string `json:"tags"`
}

type ListResult struct {
	Pages   []Summary `json:"pages"`
	Total   int       `json:"total"`
	Skipped *int      `json:"skipped,omitempty"`
	Error   string    `json:"error,omitempty"`
}

type GetRequest struct {
	Route string
	Lang  string
}

type Page struct {
	Route       string         `json:"route"`
	File        string         `json:"file"`
	Frontmatter map[string]any `json:"frontmatter"`
	Content     string         `json:"content"`
}

func New(contentRoot string) *Service {
	return &Service{ContentRoot: contentRoot, MaxPageBytes: 1 << 20}
}

func (s *Service) List(ctx context.Context, req ListRequest) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	contentRoot, err := pathguard.CanonicalDir(s.ContentRoot)
	if err != nil {
		return ListResult{}, err
	}
	section, err := normalizeUserPath(req.Section, "section")
	if err != nil {
		return ListResult{}, err
	}
	lang := strings.TrimSpace(req.Lang)
	if lang != "" {
		if _, err := validateLang(lang); err != nil {
			return ListResult{}, err
		}
	}

	scanRoot := contentRoot
	if section != "" {
		resolved, err := pathguard.ResolveExistingPath(contentRoot, section)
		if err != nil {
			return ListResult{}, err
		}
		scanRoot = resolved
	}
	if _, err := os.Stat(scanRoot); err != nil {
		if os.IsNotExist(err) {
			return ListResult{Pages: []Summary{}, Total: 0}, nil
		}
		return ListResult{Pages: []Summary{}, Total: 0, Error: err.Error()}, nil
	}

	var pages []Summary
	skipped := 0
	collect := func(abs string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(abs)
		if err != nil {
			if errorsIsNotExist(err) {
				return nil
			}
			skipped++
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed")
		}
		if limit := s.maxPageBytes(); limit > 0 && info.Size() > limit {
			skipped++
			return nil
		}
		rel, err := filepath.Rel(contentRoot, abs)
		if err != nil {
			skipped++
			return nil
		}
		fm, _, err := frontmatter.ParseFile(abs)
		if err != nil {
			skipped++
			return nil
		}
		route := routeForPage(rel)
		fileLang := langFromName(filepath.Base(abs))
		if lang != "" && fileLang != lang {
			return nil
		}
		pages = append(pages, Summary{
			Route: route,
			Lang:  fileLang,
			File:  filepath.ToSlash(rel),
			Title: stringValue(fm["title"]),
			Date:  stringValue(fm["date"]),
			Draft: boolValue(fm["draft"]),
			Tags:  stringSlice(fm["tags"]),
		})
		return nil
	}

	if err := collectDirectIndex(scanRoot, collect); err != nil {
		return ListResult{}, err
	}
	if err := collectRecursiveIndexes(scanRoot, collect); err != nil {
		return ListResult{}, err
	}

	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].Date == pages[j].Date {
			if pages[i].File == pages[j].File {
				return pages[i].Lang < pages[j].Lang
			}
			return pages[i].File < pages[j].File
		}
		return pages[i].Date > pages[j].Date
	})

	res := ListResult{Pages: pages, Total: len(pages)}
	if skipped > 0 {
		res.Skipped = &skipped
	}
	return res, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	route, root, err := normalizeRoute(req.Route)
	if err != nil {
		return Page{}, err
	}
	lang := strings.TrimSpace(req.Lang)
	if lang != "" {
		if _, err := validateLang(lang); err != nil {
			return Page{}, err
		}
	}
	rel, file, err := s.findPage(route, lang)
	if err != nil {
		return Page{}, err
	}
	if limit := s.maxPageBytes(); limit > 0 {
		info, err := os.Stat(file)
		if err != nil {
			return Page{}, err
		}
		if info.Size() > limit {
			return Page{}, fmt.Errorf("page too large: %d bytes (max %d)", info.Size(), limit)
		}
	}
	fm, content, err := frontmatter.ParseFile(file)
	if err != nil {
		return Page{}, err
	}
	if root {
		return Page{Route: "/", File: rel, Frontmatter: fm, Content: content}, nil
	}
	return Page{Route: route, File: rel, Frontmatter: fm, Content: content}, nil
}

func (s *Service) findPage(route, lang string) (string, string, error) {
	if route == "_index" {
		if lang != "" {
			candidateRel := fmt.Sprintf("_index.%s.md", lang)
			if candidate, err := pathguard.ResolveExistingPath(s.ContentRoot, candidateRel); err == nil {
				return filepath.ToSlash(candidateRel), candidate, nil
			}
		}
		candidateRel := "_index.md"
		if candidate, err := pathguard.ResolveExistingPath(s.ContentRoot, candidateRel); err == nil {
			return candidateRel, candidate, nil
		}
		return "", "", fmt.Errorf("Page not found: %s (lang=%s)", route, lang)
	}
	if lang != "" {
		candidateRel := filepath.ToSlash(filepath.Join(route, fmt.Sprintf("index.%s.md", lang)))
		if candidate, err := pathguard.ResolveExistingPath(s.ContentRoot, candidateRel); err == nil {
			return candidateRel, candidate, nil
		}
	}
	candidateRel := filepath.ToSlash(filepath.Join(route, "index.md"))
	if candidate, err := pathguard.ResolveExistingPath(s.ContentRoot, candidateRel); err == nil {
		return candidateRel, candidate, nil
	}
	return "", "", fmt.Errorf("Page not found: %s (lang=%s)", route, lang)
}

func collectDirectIndex(root string, visit func(string) error) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed")
		}
		name := entry.Name()
		if strings.HasPrefix(name, "_index.") && strings.HasSuffix(name, ".md") {
			if err := visit(filepath.Join(root, name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectRecursiveIndexes(root string, visit func(string) error) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed")
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(filepath.Base(p), "index.") && strings.HasSuffix(filepath.Base(p), ".md") && strings.Count(filepath.Base(p), ".") >= 2 {
			return visit(p)
		}
		return nil
	})
}

func normalizeRoute(route string) (string, bool, error) {
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

func normalizeUserPath(raw string, field string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if strings.Contains(s, "\\") || strings.HasPrefix(s, "/") || strings.Contains(s, "..") {
		if field == "section" {
			return "", fmt.Errorf("Invalid section: path traversal detected")
		}
		return "", fmt.Errorf("path traversal detected")
	}
	clean, err := pathguard.ValidateRelative(s)
	if err != nil {
		return "", err
	}
	return clean, nil
}

func routeForPage(rel string) string {
	rel = filepath.ToSlash(rel)
	switch {
	case rel == "_index.md":
		return "/"
	case strings.HasPrefix(rel, "_index."):
		return "/" + strings.TrimSuffix(filepath.ToSlash(filepath.Dir(rel)), ".")
	}
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		return "/"
	}
	return "/" + dir
}

func langFromName(name string) string {
	base := filepath.Base(name)
	parts := strings.Split(base, ".")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-2]
}

func validateLang(lang string) (string, error) {
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

func stringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}

func boolValue(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func stringSlice(v any) []string {
	switch x := v.(type) {
	case []string:
		if x == nil {
			return []string{}
		}
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			out = append(out, stringValue(item))
		}
		return out
	default:
		return []string{}
	}
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

func (s *Service) maxPageBytes() int64 {
	if s == nil || s.MaxPageBytes <= 0 {
		return 1 << 20
	}
	return s.MaxPageBytes
}
