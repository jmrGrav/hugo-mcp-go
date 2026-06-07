package mutations

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

type AssetService struct {
	Stage          *staging.Workspace
	MaxUploadBytes int64
}

type UploadAssetRequest struct {
	Filename  string
	Data      string
	Subfolder string
}

type UploadAssetResult struct {
	Status          string         `json:"status"`
	Path            string         `json:"path"`
	PublicURL       string         `json:"public_url"`
	SizeBytes       int            `json:"size_bytes"`
	Deploy          string         `json:"deploy"`
	HooksEnabled    bool           `json:"hooks.enabled,omitempty"`
	CloudflarePurge map[string]any `json:"cloudflare_purge,omitempty"`
	GoogleIndexing  map[string]any `json:"google_indexing,omitempty"`
	IndexNow        map[string]any `json:"indexnow,omitempty"`
	QueuedURLsCount int             `json:"queued_urls_count,omitempty"`
	FailedJobsCount int             `json:"failed_jobs_count,omitempty"`
}

func NewAssetService(ws *staging.Workspace) *AssetService {
	return &AssetService{Stage: ws, MaxUploadBytes: 10 * 1024 * 1024}
}

func (s *AssetService) UploadAsset(ctx context.Context, req UploadAssetRequest) (UploadAssetResult, error) {
	if err := ctx.Err(); err != nil {
		return UploadAssetResult{}, err
	}
	if s == nil || s.Stage == nil {
		return UploadAssetResult{}, fmt.Errorf("missing staging workspace")
	}
	if strings.TrimSpace(req.Filename) == "" {
		return UploadAssetResult{}, fmt.Errorf("missing required field: filename")
	}
	if strings.TrimSpace(req.Data) == "" {
		return UploadAssetResult{}, fmt.Errorf("missing required field: data")
	}
	filename, err := validateUploadFilename(req.Filename)
	if err != nil {
		return UploadAssetResult{}, err
	}
	subfolder, err := validateUploadSubfolder(req.Subfolder)
	if err != nil {
		return UploadAssetResult{}, err
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := uploadAssetMIMETypes[ext]; !ok {
		return UploadAssetResult{}, fmt.Errorf("Unsupported extension %q. Allowed: %s", ext, formatAllowedUploadExtensions())
	}
	maxUploadBytes := s.MaxUploadBytes
	if maxUploadBytes <= 0 {
		maxUploadBytes = 10 * 1024 * 1024
	}
	if int64(base64.StdEncoding.DecodedLen(len(req.Data))) > maxUploadBytes {
		return UploadAssetResult{}, fmt.Errorf("File too large: decoded size exceeds max %d bytes", maxUploadBytes)
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(req.Data)
	if err != nil {
		return UploadAssetResult{}, fmt.Errorf("Invalid base64 data")
	}
	if int64(len(raw)) > maxUploadBytes {
		return UploadAssetResult{}, fmt.Errorf("File too large: %d bytes (max %d)", len(raw), maxUploadBytes)
	}
	rel := filepath.ToSlash(filepath.Join(subfolder, filename))
	if _, err := s.Stage.ResolveNewStatic(rel); err != nil {
		return UploadAssetResult{}, err
	}
	if err := writeBytesAt(s.Stage.StaticRoot, rel, raw, true); err != nil {
		return UploadAssetResult{}, err
	}
	target := filepath.Join(s.Stage.StaticRoot, filepath.FromSlash(rel))
	finalRel, err := filepath.Rel(s.Stage.HugoRoot, target)
	if err != nil {
		return UploadAssetResult{}, err
	}
	return UploadAssetResult{
		Status:    "ok",
		Path:      filepath.ToSlash(finalRel),
		PublicURL: "/" + filepath.ToSlash(filepath.Join(subfolder, filename)),
		SizeBytes: len(raw),
		Deploy:    "DEPLOY_SKIPPED",
	}, nil
}

var uploadAssetMIMETypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".gif":  "image/gif",
}

func validateUploadFilename(name string) (string, error) {
	s := strings.TrimSpace(name)
	if s == "" {
		return "", fmt.Errorf("Invalid filename: must not contain /, backslash, or ..")
	}
	if strings.Contains(s, "/") || strings.Contains(s, "\\") || strings.Contains(s, "..") {
		return "", fmt.Errorf("Invalid filename: must not contain /, backslash, or ..")
	}
	return s, nil
}

func validateUploadSubfolder(subfolder string) (string, error) {
	s := strings.TrimSpace(subfolder)
	if s == "" {
		return "images", nil
	}
	if strings.Contains(s, "..") || strings.HasPrefix(s, "/") || strings.Contains(s, "\\") {
		return "", fmt.Errorf("Invalid subfolder: must be relative without ..")
	}
	clean, err := pathguard.ValidateRelative(s)
	if err != nil {
		return "", fmt.Errorf("Invalid subfolder: must be relative without ..")
	}
	return clean, nil
}

func uploadAssetExtensionList() []string {
	return []string{".gif", ".jpeg", ".jpg", ".png", ".svg", ".webp"}
}

func formatAllowedUploadExtensions() string {
	exts := uploadAssetExtensionList()
	out := "['" + exts[0] + "'"
	for i := 1; i < len(exts); i++ {
		out += ", '" + exts[i] + "'"
	}
	out += "]"
	return out
}
