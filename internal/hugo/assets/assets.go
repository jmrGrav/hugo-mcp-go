package assets

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

type Service struct {
	HugoRoot    string
	ContentRoot string
	StaticRoot  string
}

type ListRequest struct {
	Type       string
	PathPrefix string
	MaxResults int
}

type Asset struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
	Modified  string `json:"modified,omitempty"`
}

type ListResult struct {
	Count     int     `json:"count"`
	Truncated bool    `json:"truncated"`
	Assets    []Asset `json:"assets"`
}

func New(hugoRoot, contentRoot, staticRoot string) *Service {
	return &Service{HugoRoot: hugoRoot, ContentRoot: contentRoot, StaticRoot: staticRoot}
}

func (s *Service) List(ctx context.Context, req ListRequest) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	hugoRoot, err := pathguard.CanonicalDir(s.HugoRoot)
	if err != nil {
		return ListResult{}, err
	}
	assetType := strings.TrimSpace(req.Type)
	if assetType == "" {
		assetType = "all"
	}
	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 100
	}
	if maxResults > 500 {
		maxResults = 500
	}
	pathPrefix, err := normalizePathPrefix(req.PathPrefix)
	if err != nil {
		return ListResult{}, err
	}
	exts := extensionsForType(assetType)
	if exts == nil {
		return ListResult{}, fmt.Errorf("unsupported asset type: %s", assetType)
	}
	out := make([]Asset, 0, maxResults)
	for _, root := range []string{s.StaticRoot, s.ContentRoot} {
		if err := ctx.Err(); err != nil {
			return ListResult{}, err
		}
		resolved, err := pathguard.ResolveScanRoot(root, pathPrefix)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ListResult{}, err
		}
		if _, err := os.Stat(resolved); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ListResult{}, err
		}
		if err := filepath.WalkDir(resolved, func(p string, d fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlinks are not allowed")
			}
			if d.IsDir() {
				return nil
			}
			if !matchesType(filepath.Base(p), exts) {
				return nil
			}
			if strings.HasPrefix(filepath.Base(p), "index.") || strings.HasPrefix(filepath.Base(p), "_index.") {
				return nil
			}
			info, err := os.Stat(p)
			if err != nil {
				return nil
			}
			rel, err := filepath.Rel(hugoRoot, p)
			if err != nil {
				return nil
			}
			out = append(out, Asset{
				Path:      filepath.ToSlash(rel),
				SizeBytes: info.Size(),
				MimeType:  mimeTypeFor(filepath.Ext(p)),
				Modified:  info.ModTime().Local().Format("2006-01-02T15:04:05"),
			})
			return nil
		}); err != nil {
			return ListResult{}, err
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Modified == out[j].Modified {
			return out[i].Path < out[j].Path
		}
		return out[i].Modified > out[j].Modified
	})
	truncated := len(out) > maxResults
	if truncated {
		out = out[:maxResults]
	}
	return ListResult{Count: len(out), Truncated: truncated, Assets: out}, nil
}

func normalizePathPrefix(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if strings.Contains(s, "\\") || strings.HasPrefix(s, "/") || strings.Contains(s, "..") {
		return "", fmt.Errorf("Invalid path_prefix: must be relative without '..'")
	}
	clean, err := pathguard.ValidateRelative(s)
	if err != nil {
		return "", fmt.Errorf("Invalid path_prefix: must be relative without '..'")
	}
	return clean, nil
}

func extensionsForType(typ string) map[string]struct{} {
	switch typ {
	case "image":
		return map[string]struct{}{".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {}, ".svg": {}, ".avif": {}}
	case "document":
		return map[string]struct{}{".pdf": {}, ".txt": {}, ".csv": {}, ".zip": {}}
	case "all":
		out := map[string]struct{}{}
		for k := range extensionsForType("image") {
			out[k] = struct{}{}
		}
		for k := range extensionsForType("document") {
			out[k] = struct{}{}
		}
		return out
	default:
		return nil
	}
}

func matchesType(name string, exts map[string]struct{}) bool {
	_, ok := exts[strings.ToLower(filepath.Ext(name))]
	return ok
}

func mimeTypeFor(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".avif":
		return "image/avif"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func stableNow() string { return time.Now().Format(time.RFC3339) }
