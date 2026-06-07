package staging

import (
	"fmt"

	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
)

type Workspace struct {
	HugoRoot    string
	ContentRoot string
	StaticRoot  string
	PublicRoot  string
	WorkRoot    string
}

func New(hugoRoot, contentRoot, staticRoot, publicRoot, workRoot string) (*Workspace, error) {
	hugoRoot, err := pathguard.CanonicalDir(hugoRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid staging HUGO_ROOT: %w", err)
	}
	contentRoot, err = validatedChild(hugoRoot, contentRoot, "CONTENT_ROOT")
	if err != nil {
		return nil, err
	}
	staticRoot, err = validatedChild(hugoRoot, staticRoot, "STATIC_ROOT")
	if err != nil {
		return nil, err
	}
	publicRoot, err = validatedChild(hugoRoot, publicRoot, "PUBLIC_ROOT")
	if err != nil {
		return nil, err
	}
	workRoot, err = validatedChild(hugoRoot, workRoot, "WORK_ROOT")
	if err != nil {
		return nil, err
	}
	if err := ensureDistinct(contentRoot, staticRoot, publicRoot, workRoot); err != nil {
		return nil, err
	}
	return &Workspace{
		HugoRoot:    hugoRoot,
		ContentRoot: contentRoot,
		StaticRoot:  staticRoot,
		PublicRoot:  publicRoot,
		WorkRoot:    workRoot,
	}, nil
}

func validatedChild(hugoRoot, root, field string) (string, error) {
	canonical, err := pathguard.CanonicalDir(root)
	if err != nil {
		return "", fmt.Errorf("invalid staging %s: %w", field, err)
	}
	if !pathguard.WithinRoot(hugoRoot, canonical) {
		return "", fmt.Errorf("invalid staging %s: must be inside HUGO_ROOT", field)
	}
	return canonical, nil
}

func ensureDistinct(roots ...string) error {
	for i := 0; i < len(roots); i++ {
		for j := i + 1; j < len(roots); j++ {
			if roots[i] == roots[j] {
				return fmt.Errorf("staging roots must be distinct")
			}
		}
	}
	return nil
}
