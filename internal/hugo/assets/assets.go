package assets

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	Cursor     string
}

type Asset struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
	Modified  string `json:"modified,omitempty"`
}

type ListResult struct {
	Count      int     `json:"count"`
	Truncated  bool    `json:"truncated"`
	Assets     []Asset `json:"assets"`
	HasMore    bool    `json:"has_more,omitempty"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

type ChunkRequest struct {
	Path       string
	Cursor     int
	ChunkBytes int
}

type ChunkResult struct {
	Path       string `json:"path"`
	MimeType   string `json:"mime_type"`
	Chunk      string `json:"chunk"`
	Cursor     int    `json:"cursor"`
	NextCursor string `json:"next_cursor,omitempty"`
	ChunkBytes int    `json:"chunk_bytes"`
	TotalBytes int    `json:"total_bytes"`
	IsLast     bool   `json:"is_last"`
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
	offset, err := parseOffset(req.Cursor)
	if err != nil {
		return ListResult{}, err
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
	total := len(out)
	if offset > total {
		offset = len(out)
	}
	end := offset + maxResults
	if end > total {
		end = total
	}
	window := out[offset:end]
	truncated := end < total
	res := ListResult{Count: len(window), Truncated: truncated, Assets: window}
	if truncated {
		res.HasMore = true
		res.NextCursor = strconv.Itoa(end)
	}
	return res, nil
}

func (s *Service) GetChunk(ctx context.Context, req ChunkRequest) (ChunkResult, error) {
	if err := ctx.Err(); err != nil {
		return ChunkResult{}, err
	}
	hugoRoot, err := pathguard.CanonicalDir(s.HugoRoot)
	if err != nil {
		return ChunkResult{}, err
	}
	rel, err := normalizeChunkPath(req.Path)
	if err != nil {
		return ChunkResult{}, err
	}
	file, err := pathguard.ResolveExistingPath(hugoRoot, rel)
	if err != nil {
		return ChunkResult{}, err
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return ChunkResult{}, err
	}
	if req.ChunkBytes <= 0 {
		req.ChunkBytes = 64 << 10
	}
	if req.Cursor < 0 {
		return ChunkResult{}, fmt.Errorf("invalid cursor: must be >= 0")
	}
	offset := req.Cursor
	if offset > len(raw) {
		offset = len(raw)
	}
	end := offset + req.ChunkBytes
	if end > len(raw) {
		end = len(raw)
	}
	chunk := base64.StdEncoding.EncodeToString(raw[offset:end])
	out := ChunkResult{
		Path:       filepath.ToSlash(rel),
		MimeType:   mimeTypeFor(filepath.Ext(rel)),
		Chunk:      chunk,
		Cursor:     offset,
		ChunkBytes: end - offset,
		TotalBytes: len(raw),
		IsLast:     end >= len(raw),
	}
	if !out.IsLast {
		out.NextCursor = strconv.Itoa(end)
	}
	return out, nil
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

func parseOffset(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor: %w", err)
	}
	if v < 0 {
		return 0, fmt.Errorf("invalid cursor: must be >= 0")
	}
	return v, nil
}

func normalizeChunkPath(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("missing path")
	}
	if strings.Contains(s, "\\") || strings.Contains(s, "..") || strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("invalid path")
	}
	return pathguard.ValidateRelative(s)
}
