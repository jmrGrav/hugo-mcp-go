package sri

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
	"github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard"
	"gopkg.in/yaml.v3"
)

const pluginName = "sri-check"

type Option func(*Service)

type Service struct {
	cfg        Config
	httpClient *http.Client
	hooks      hookSummaryProvider
	builder    Builder
	now        func() time.Time
}

func NewService(cfg Config, opts ...Option) (*Service, error) {
	normalized, err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg: normalized,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		now: time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if s.httpClient == nil {
		s.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if s.now == nil {
		s.now = time.Now
	}
	return s, nil
}

func WithHTTPClient(client *http.Client) Option {
	return func(s *Service) {
		if client != nil {
			s.httpClient = client
		}
	}
}

func WithHooks(processor hookSummaryProvider) Option {
	return func(s *Service) {
		s.hooks = processor
	}
}

func WithBuilder(builder Builder) Option {
	return func(s *Service) {
		s.builder = builder
	}
}

func (s *Service) Check(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if s == nil {
		return Result{}, errors.New("sri service not configured")
	}

	requestedAutoFix := boolValue(req.AutoFix)
	dryRun := boolValueDefault(req.DryRun, s.cfg.DryRunDefault)
	if !s.cfg.Enabled {
		return Result{
			Plugin:           pluginName,
			Success:          false,
			ExitCode:         2,
			AutoFixRequested: requestedAutoFix,
			DryRun:           dryRun,
			Report: Report{
				Exit:    2,
				Summary: "DISABLED",
				AutoFix: AutoFixReport{
					Skipped: true,
				},
				Incident: IncidentReport{
					Resolved: []string{},
				},
				DryRun: dryRun,
			},
		}, nil
	}

	scan := newRunState()
	scan.Other = append(scan.Other, s.scanRoots(ctx, scan)...)
	cdnPath, sriPath, fileWarnings := s.locateDataFiles()
	scan.Other = append(scan.Other, fileWarnings...)

	cdnEntries, cdnWarnings, err := parseCDNConfig(cdnPath)
	if err != nil {
		scan.Other = append(scan.Other, redactScanError(err))
	}
	scan.Other = append(scan.Other, cdnWarnings...)

	sriEntries, sriWarnings, err := parseSRIConfig(sriPath)
	if err != nil {
		scan.Other = append(scan.Other, redactScanError(err))
	}
	scan.Other = append(scan.Other, sriWarnings...)

	if len(cdnEntries) > 0 {
		s.evaluateVersions(ctx, scan, cdnEntries)
	}
	s.evaluateSRI(ctx, scan, sriEntries)

	result := scan.toResult(requestedAutoFix, dryRun)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	if requestedAutoFix && !dryRun && len(scan.AutoFixCandidates) > 0 && cdnPath != "" && sriPath != "" {
		fixed, downstream, fixWarnings := s.applyAutoFix(ctx, cdnPath, sriPath, scan, cdnEntries, sriEntries)
		result.Report.Diagnostic.Other = append(result.Report.Diagnostic.Other, fixWarnings...)
		if fixed {
			result.Report.AutoFix.Ran = true
			result.Report.AutoFix.Applied = append([]string(nil), scan.appliedDescriptions()...)
			result.Report.AutoFix.Skipped = false
			if downstream != nil {
				result.Downstream = downstream
				if cloudflare, ok := downstream["cloudflare_purge"].(map[string]any); ok {
					status, _ := cloudflare["status"].(string)
					result.Report.AutoFix.CFPurged = status != ""
				}
			}
		} else if len(scan.AutoFixCandidates) > 0 {
			result.Report.AutoFix.Failed = append(result.Report.AutoFix.Failed, scan.AutoFixFailures...)
		}

		post := newRunState()
		post.Other = append(post.Other, s.scanRoots(ctx, post)...)
		cdnEntries, cdnWarnings, err = parseCDNConfig(cdnPath)
		if err != nil {
			post.Other = append(post.Other, redactScanError(err))
		}
		post.Other = append(post.Other, cdnWarnings...)
		sriEntries, sriWarnings, err = parseSRIConfig(sriPath)
		if err != nil {
			post.Other = append(post.Other, redactScanError(err))
		}
		post.Other = append(post.Other, sriWarnings...)
		if len(cdnEntries) > 0 {
			s.evaluateVersions(ctx, post, cdnEntries)
		}
		s.evaluateSRI(ctx, post, sriEntries)
		if len(post.HashMismatch) > 0 || len(post.MajorOutdated) > 0 || len(post.Other) > 0 {
			result.Report.Diagnostic.HashMismatch = post.HashMismatch
			result.Report.Diagnostic.MajorOutdated = post.MajorOutdated
			result.Report.Diagnostic.MinorOutdated = post.MinorOutdated
			result.Report.Diagnostic.Other = append(result.Report.Diagnostic.Other, post.Other...)
		} else {
			result.Report.Diagnostic.HashMismatch = nil
			result.Report.Diagnostic.MajorOutdated = nil
			result.Report.Diagnostic.MinorOutdated = "0"
			result.Report.Diagnostic.Other = append([]string(nil), result.Report.Diagnostic.Other...)
		}
	}

	result.Report.DryRun = dryRun
	result.Report.Incident = IncidentReport{Resolved: []string{}}
	if len(result.Report.AutoFix.Applied) == 0 && requestedAutoFix && dryRun {
		result.Report.AutoFix.Skipped = true
	}
	finalizeResult(&result)
	return result, nil
}

type runState struct {
	HashMismatch      []string
	MajorOutdated     []string
	MinorOutdated     string
	Other             []string
	ActiveRefs        map[string]activeRef
	CDNEntries        map[string]cdnEntry
	SRIEntries        map[string]string
	AutoFixCandidates []fixCandidate
	AutoFixFailures   []string
	AppliedFixes      []fixCandidate
}

func newRunState() *runState {
	return &runState{
		ActiveRefs: make(map[string]activeRef),
		CDNEntries: make(map[string]cdnEntry),
		SRIEntries: make(map[string]string),
	}
}

func (s *Service) evaluateVersions(ctx context.Context, state *runState, entries map[string]cdnEntry) {
	activeByPkg := state.ActiveRefs
	for key, entry := range entries {
		state.CDNEntries[key] = entry
		if _, ok := activeByPkg[entry.Package]; !ok {
			continue
		}
		latest, err := s.fetchLatestVersion(ctx, entry.Package)
		if err != nil {
			state.Other = append(state.Other, fmt.Sprintf("API FAIL: %s", entry.Package))
			continue
		}
		if latest == "" {
			state.Other = append(state.Other, fmt.Sprintf("API FAIL: %s", entry.Package))
			continue
		}
		if entry.Version == latest {
			continue
		}
		if compareVersions(entry.Version, latest) >= 0 {
			continue
		}
		if sameMajor(entry.Version, latest) {
			state.AutoFixCandidates = append(state.AutoFixCandidates, fixCandidate{
				Key:     key,
				Package: entry.Package,
				Old:     entry.Version,
				New:     latest,
			})
			continue
		}
		state.MajorOutdated = append(state.MajorOutdated, fmt.Sprintf("OUTDATED MAJOR: %s %s → %s (manual review)", entry.Package, entry.Version, latest))
	}
	state.MinorOutdated = strconv.Itoa(len(state.AutoFixCandidates))
}

func (s *Service) evaluateSRI(ctx context.Context, state *runState, entries map[string]string) {
	for urlString, stored := range entries {
		state.SRIEntries[urlString] = stored
		if !strings.HasPrefix(urlString, "https://") {
			continue
		}
		parsed, err := url.Parse(urlString)
		if err != nil {
			state.Other = append(state.Other, fmt.Sprintf("invalid SRI url: %s", urlString))
			continue
		}
		if !hostAllowed(strings.ToLower(parsed.Host), s.cfg.AllowedCDNHosts) {
			state.Other = append(state.Other, fmt.Sprintf("host not allowlisted: %s", parsed.Host))
			continue
		}
		body, err := s.fetchBytes(ctx, urlString)
		if err != nil {
			state.Other = append(state.Other, fmt.Sprintf("FETCH FAIL: %s", urlString))
			continue
		}
		live := "sha256-" + base64.StdEncoding.EncodeToString(sha256Sum(body))
		if live != stored {
			state.HashMismatch = append(state.HashMismatch, fmt.Sprintf("HASH MISMATCH: %s (stocké=%s live=%s)", urlString, stored, live))
		}
	}

	for _, ref := range state.ActiveRefs {
		if _, ok := entries[ref.URL]; ok {
			continue
		}
		state.Other = append(state.Other, fmt.Sprintf("MISSING SRI ENTRY: %s", ref.URL))
	}
}

func (r *runState) toResult(requestedAutoFix, dryRun bool) Result {
	result := Result{
		Plugin:           pluginName,
		AutoFixRequested: requestedAutoFix,
		DryRun:           dryRun,
		Report: Report{
			Diagnostic: DiagnosticReport{
				HashMismatch:  append([]string(nil), r.HashMismatch...),
				MajorOutdated: append([]string(nil), r.MajorOutdated...),
				MinorOutdated: r.MinorOutdated,
				Other:         append([]string(nil), r.Other...),
			},
			AutoFix: AutoFixReport{
				Skipped: !requestedAutoFix || dryRun,
			},
			Incident: IncidentReport{
				Resolved: []string{},
			},
			DryRun: dryRun,
		},
	}
	if result.Report.Diagnostic.MinorOutdated == "" {
		result.Report.Diagnostic.MinorOutdated = "0"
	}
	return result
}

func (r *runState) appliedDescriptions() []string {
	out := make([]string, 0, len(r.AppliedFixes))
	for _, cand := range r.AppliedFixes {
		out = append(out, fmt.Sprintf("%s %s → %s", cand.Package, cand.Old, cand.New))
	}
	return out
}

func finalizeResult(result *Result) {
	if result == nil {
		return
	}
	warn := len(result.Report.Diagnostic.HashMismatch) > 0 ||
		len(result.Report.Diagnostic.MajorOutdated) > 0 ||
		len(result.Report.Diagnostic.Other) > 0
	switch result.Report.Summary {
	case "DISABLED":
		result.Success = false
		result.Report.Exit = result.ExitCode
		return
	default:
		if warn {
			result.Report.Summary = "WARN"
			result.Report.Exit = 1
			result.ExitCode = 1
			result.Success = false
			return
		}
		result.Report.Summary = "OK"
		result.Report.Exit = 0
		result.ExitCode = 0
		result.Success = true
	}
}

type activeRef struct {
	URL     string
	Package string
	Version string
	Host    string
	Source  string
}

type cdnEntry struct {
	Key      string
	Package  string
	Version  string
	Path     string
	RawValue string
}

type fixCandidate struct {
	Key     string
	Package string
	Old     string
	New     string
}

var cdnURLRe = regexp.MustCompile(`https://([^/]+)/npm/((?:@[^/]+/)?[^@/]+)@([^/]+)/([^"'` + "`" + `\s<>]+)`)
var jsDelivrValueRe = regexp.MustCompile(`^((?:@[^/]+/)?[^@/]+)@([^/]+)/(.+)$`)

func (s *Service) scanRoots(ctx context.Context, state *runState) []string {
	warnings := []string{}
	roots := s.cfg.ScanRoots
	if len(roots) == 0 {
		return warnings
	}
	seenFiles := map[string]struct{}{}
	for _, relRoot := range roots {
		if err := ctx.Err(); err != nil {
			warnings = append(warnings, err.Error())
			return warnings
		}
		cleanRel, err := pathguard.ValidateRelative(relRoot)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("scan root %q: %v", relRoot, err))
			continue
		}
		if cleanRel == "" {
			continue
		}
		rootPath := filepath.Join(s.cfg.HugoRoot, filepath.FromSlash(cleanRel))
		info, err := os.Stat(rootPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			warnings = append(warnings, fmt.Sprintf("scan root %q: %v", relRoot, err))
			continue
		}
		if !info.IsDir() {
			warnings = append(warnings, fmt.Sprintf("scan root %q: not a directory", relRoot))
			continue
		}
		count := 0
		err = pathguard.WalkDirNoSymlink(rootPath, func(path string, d fs.DirEntry) error {
			if d.IsDir() {
				return nil
			}
			if count >= s.cfg.MaxFiles {
				warnings = append(warnings, fmt.Sprintf("scan file limit reached at %d files", s.cfg.MaxFiles))
				return fs.SkipAll
			}
			rel, err := filepath.Rel(s.cfg.HugoRoot, path)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("scan path rel: %v", err))
				return nil
			}
			rel = filepath.ToSlash(rel)
			if _, ok := seenFiles[rel]; ok {
				return nil
			}
			seenFiles[rel] = struct{}{}
			count++
			if !isTextLike(rel) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("scan read %s: %v", rel, err))
				return nil
			}
			if int64(len(data)) > s.cfg.MaxFileBytes {
				warnings = append(warnings, fmt.Sprintf("scan file too large: %s", rel))
				return nil
			}
			matches := cdnURLRe.FindAllSubmatch(data, -1)
			for _, match := range matches {
				host := strings.ToLower(string(match[1]))
				if !hostAllowed(host, s.cfg.AllowedCDNHosts) {
					warnings = append(warnings, fmt.Sprintf("host not allowlisted: %s", host))
					continue
				}
				pkg := string(match[2])
				version := string(match[3])
				url := "https://" + host + "/npm/" + pkg + "@" + version + "/" + string(match[4])
				prev, ok := state.ActiveRefs[pkg]
				if ok && prev.Version != version {
					warnings = append(warnings, fmt.Sprintf("conflicting versions for %s: %s vs %s", pkg, prev.Version, version))
					continue
				}
				state.ActiveRefs[pkg] = activeRef{
					URL:     url,
					Package: pkg,
					Version: version,
					Host:    host,
					Source:  rel,
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, fs.SkipAll) {
			warnings = append(warnings, fmt.Sprintf("scan root %q: %v", relRoot, err))
		}
	}
	return warnings
}

func (s *Service) locateDataFiles() (string, string, []string) {
	var warnings []string
	cdnPath, err := findFileBySuffix(s.cfg.HugoRoot, filepath.FromSlash("assets/data/cdn/jsdelivr.yml"))
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("cdn file: %v", err))
	}
	sriPath, err := findFileBySuffix(s.cfg.HugoRoot, filepath.FromSlash("data/sri.yaml"))
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("sri file: %v", err))
	}
	return cdnPath, sriPath, warnings
}

func findFileBySuffix(root, suffix string) (string, error) {
	var found string
	suffix = filepath.ToSlash(strings.TrimSpace(suffix))
	err := pathguard.WalkDirNoSymlink(root, func(path string, d fs.DirEntry) error {
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(filepath.ToSlash(rel), suffix) {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("not found: %s", filepath.ToSlash(suffix))
	}
	return found, nil
}

func parseCDNConfig(path string) (map[string]cdnEntry, []string, error) {
	out := map[string]cdnEntry{}
	if path == "" {
		return out, nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, nil, err
	}
	var parsed struct {
		LibFiles map[string]string `yaml:"libFiles"`
	}
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		return out, nil, err
	}
	for key, value := range parsed.LibFiles {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		m := jsDelivrValueRe.FindStringSubmatch(value)
		if len(m) != 4 {
			continue
		}
		out[key] = cdnEntry{
			Key:      key,
			Package:  m[1],
			Version:  m[2],
			Path:     m[3],
			RawValue: value,
		}
	}
	return out, nil, nil
}

func parseSRIConfig(path string) (map[string]string, []string, error) {
	out := map[string]string{}
	if path == "" {
		return out, nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return out, nil, nil
	}
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return out, nil, err
	}
	return out, nil, nil
}

func (s *Service) fetchLatestVersion(ctx context.Context, pkg string) (string, error) {
	endpoint := "https://data.jsdelivr.com/v1/packages/npm/" + url.PathEscape(pkg)
	body, err := s.fetchBytes(ctx, endpoint)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Tags struct {
			Latest string `json:"latest"`
		} `json:"tags"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	return strings.TrimSpace(parsed.Tags.Latest), nil
}

func (s *Service) fetchBytes(ctx context.Context, target string) ([]byte, error) {
	if s.httpClient == nil {
		return nil, errors.New("http client not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func (s *Service) applyAutoFix(ctx context.Context, cdnPath, sriPath string, scan *runState, cdnEntries map[string]cdnEntry, sriEntries map[string]string) (bool, map[string]any, []string) {
	_ = ctx
	if len(scan.AutoFixCandidates) == 0 {
		return false, nil, nil
	}
	originalCDN, err := os.ReadFile(cdnPath)
	if err != nil {
		return false, nil, []string{fmt.Sprintf("autofix read cdn: %v", err)}
	}
	originalSRI, err := os.ReadFile(sriPath)
	if err != nil {
		return false, nil, []string{fmt.Sprintf("autofix read sri: %v", err)}
	}
	cdnRaw := string(originalCDN)
	var sriMap map[string]string
	if err := yaml.Unmarshal(originalSRI, &sriMap); err != nil {
		return false, nil, []string{fmt.Sprintf("autofix parse sri: %v", err)}
	}
	if sriMap == nil {
		sriMap = map[string]string{}
	}
	for _, cand := range scan.AutoFixCandidates {
		next := cand.New
		if next == "" {
			continue
		}
		cdnRaw = strings.ReplaceAll(cdnRaw, cand.Package+"@"+cand.Old+"/", cand.Package+"@"+next+"/")
		cdnRaw = strings.ReplaceAll(cdnRaw, "# "+cand.Package+"@"+cand.Old, "# "+cand.Package+"@"+next)
		var matchedURLs []string
		for urlString := range sriEntries {
			if strings.Contains(urlString, "/"+cand.Package+"@"+cand.Old+"/") {
				matchedURLs = append(matchedURLs, urlString)
			}
		}
		for _, oldURL := range matchedURLs {
			newURL := strings.ReplaceAll(oldURL, "@"+cand.Old+"/", "@"+next+"/")
			body, err := s.fetchBytes(ctx, newURL)
			if err != nil {
				scan.AutoFixFailures = append(scan.AutoFixFailures, fmt.Sprintf("%s %s → %s (fetch fail %s)", cand.Package, cand.Old, next, newURL))
				return false, nil, []string{fmt.Sprintf("autofix fetch fail: %s", newURL)}
			}
			live := "sha256-" + base64.StdEncoding.EncodeToString(sha256Sum(body))
			delete(sriMap, oldURL)
			sriMap[newURL] = live
		}
		scan.AppliedFixes = append(scan.AppliedFixes, cand)
	}
	if err := os.WriteFile(cdnPath, []byte(cdnRaw), 0o644); err != nil {
		return false, nil, []string{fmt.Sprintf("autofix write cdn: %v", err)}
	}
	sriOut, err := yaml.Marshal(sriMap)
	if err != nil {
		_ = os.WriteFile(cdnPath, originalCDN, 0o644)
		return false, nil, []string{fmt.Sprintf("autofix marshal sri: %v", err)}
	}
	if err := os.WriteFile(sriPath, sriOut, 0o644); err != nil {
		_ = os.WriteFile(cdnPath, originalCDN, 0o644)
		return false, nil, []string{fmt.Sprintf("autofix write sri: %v", err)}
	}
	if s.builder != nil {
		if _, err := s.builder.Build(ctx, mutations.BuildRequest{PurgeCF: false}); err != nil {
			_ = os.WriteFile(cdnPath, originalCDN, 0o644)
			_ = os.WriteFile(sriPath, originalSRI, 0o644)
			return false, nil, []string{fmt.Sprintf("autofix build failed: %v", err)}
		}
	}
	if s.cfg.TriggerHooksOnFix && s.hooks != nil && s.cfg.SiteBaseURL != "" {
		rootURL := joinBaseURL(s.cfg.SiteBaseURL, "/")
		summary := s.hooks.Process(ctx, hooks.HookEvent{
			Mutation: "sri-check.autofix",
			Action:   "URL_UPDATED",
			URLs:     []string{rootURL},
		})
		return true, summary.MCP(), nil
	}
	return true, nil, nil
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func boolValueDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func sameMajor(a, b string) bool {
	return majorPart(a) == majorPart(b)
}

func majorPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if idx := strings.IndexByte(v, '.'); idx > 0 {
		return v[:idx]
	}
	return v
}

func compareVersions(a, b string) int {
	ap := splitVersionParts(a)
	bp := splitVersionParts(b)
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(ap) {
			ai = ap[i]
		}
		if i < len(bp) {
			bi = bp[i]
		}
		switch {
		case ai < bi:
			return -1
		case ai > bi:
			return 1
		}
	}
	return 0
}

func splitVersionParts(v string) []int {
	parts := strings.Split(strings.TrimSpace(v), ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out = append(out, n)
	}
	return out
}

func sha256Sum(body []byte) []byte {
	sum := sha256.Sum256(body)
	return sum[:]
}

func hostAllowed(host string, allowed []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, candidate := range allowed {
		if host == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func isTextLike(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".md", ".txt", ".xml", ".yaml", ".yml", ".toml", ".json", ".js", ".css", ".svg":
		return true
	default:
		return false
	}
}

func joinBaseURL(base, rel string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	rel = "/" + strings.Trim(strings.TrimSpace(rel), "/")
	if rel == "/" {
		return base + "/"
	}
	return base + rel
}

func redactScanError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
