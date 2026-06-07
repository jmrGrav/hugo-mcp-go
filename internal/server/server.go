package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmrGrav/hugo-mcp-go/internal/config"
	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/assets"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/pages"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
	"github.com/jmrGrav/hugo-mcp-go/internal/runner"
	"github.com/jmrGrav/hugo-mcp-go/internal/sri"
	"github.com/jmrGrav/hugo-mcp-go/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	Name    = "hugo-mcp"
	Version = "1.0.0"
)

type Service struct {
	server     *mcp.Server
	hooksStore *hooks.Store
}

func New(cfg config.Config) (*Service, error) {
	validated, err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	publicRoot := filepath.Join(validated.HugoRoot, "public")
	workRoot := filepath.Join(validated.HugoRoot, "work")
	if err := os.MkdirAll(publicRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create public root: %w", err)
	}
	if err := os.MkdirAll(workRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create work root: %w", err)
	}
	ws, err := staging.New(validated.HugoRoot, validated.ContentRoot, validated.StaticRoot, publicRoot, workRoot)
	if err != nil {
		return nil, err
	}
	pagesSvc := pages.New(validated.ContentRoot)
	pagesSvc.MaxPageBytes = validated.MaxPageBytes
	assetsSvc := assets.New(validated.HugoRoot, validated.ContentRoot, validated.StaticRoot)
	pageMutations := mutations.NewPageService(ws)
	pageMutations.MaxPageBytes = validated.MaxPageBytes
	assetMutations := mutations.NewAssetService(ws)
	assetMutations.MaxUploadBytes = validated.MaxAssetBytes
	buildSvc := mutations.NewBuildService(ws, runner.ExecRunner{})
	hookCfg, err := hooks.LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}
	hookStore, err := hooks.OpenStore(hookCfg.HooksDB)
	if err != nil {
		return nil, err
	}
	hookPipeline := hooks.NewPipeline(hookCfg, hookStore,
		hooks.NewCloudflareClient(hookCfg, nil),
		hooks.NewGoogleIndexingClient(hookCfg, nil),
		hooks.NewIndexNowClient(hookCfg, nil),
	)
	sriCfg, err := sri.LoadConfigFromEnv(validated.HugoRoot, hookCfg.SiteBaseURL)
	if err != nil {
		return nil, err
	}
	sriSvc, err := sri.NewService(sriCfg, sri.WithHooks(hookPipeline), sri.WithBuilder(buildSvc))
	if err != nil {
		return nil, err
	}
	s := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, nil)
	tools.Register(s, tools.Deps{
		Pages:             pagesSvc,
		Assets:            assetsSvc,
		PageMutations:     pageMutations,
		AssetMutations:    assetMutations,
		Build:             buildSvc,
		Sri:               sriSvc,
		Hooks:             hookPipeline,
		HooksStore:        hookStore,
		HooksAdminEnabled: hookCfg.HooksAdminEnabled,
		SiteBaseURL:       hookCfg.SiteBaseURL,
	})
	return &Service{server: s, hooksStore: hookStore}, nil
}

func (s *Service) RunStdio(ctx context.Context) error {
	if s == nil || s.server == nil {
		return fmt.Errorf("server not initialized")
	}
	if s.hooksStore != nil {
		defer s.hooksStore.Close()
	}
	return s.server.Run(ctx, &mcp.StdioTransport{})
}

func (s *Service) MCP() *mcp.Server { return s.server }
