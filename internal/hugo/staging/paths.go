package staging

import "github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"

func (w *Workspace) ResolveExistingContent(rel string) (string, error) {
	return pathguard.ResolveExistingPath(w.ContentRoot, rel)
}

func (w *Workspace) ResolveNewContent(rel string) (string, error) {
	return pathguard.ResolveNewTargetPath(w.ContentRoot, rel)
}

func (w *Workspace) ResolveExistingStatic(rel string) (string, error) {
	return pathguard.ResolveExistingPath(w.StaticRoot, rel)
}

func (w *Workspace) ResolveNewStatic(rel string) (string, error) {
	return pathguard.ResolveNewTargetPath(w.StaticRoot, rel)
}

func (w *Workspace) ResolveExistingPublic(rel string) (string, error) {
	return pathguard.ResolveExistingPath(w.PublicRoot, rel)
}

func (w *Workspace) ResolveNewPublic(rel string) (string, error) {
	return pathguard.ResolveNewTargetPath(w.PublicRoot, rel)
}

func (w *Workspace) ResolveExistingWork(rel string) (string, error) {
	return pathguard.ResolveExistingPath(w.WorkRoot, rel)
}

func (w *Workspace) ResolveNewWork(rel string) (string, error) {
	return pathguard.ResolveNewTargetPath(w.WorkRoot, rel)
}
