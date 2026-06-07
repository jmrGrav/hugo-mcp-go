package tools

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/pages"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
	"github.com/jmrGrav/hugo-mcp-go/internal/sri"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type checkSriVersionsInput struct {
	AutoFix *bool `json:"auto_fix,omitempty"`
	DryRun  *bool `json:"dry_run,omitempty"`
}

type checkSriVersionsOutput = sri.Result

type generateFeaturedImageInput struct {
	Style    string   `json:"style,omitempty"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Accent   string   `json:"accent,omitempty"`
	Slug     string   `json:"slug"`
	Route    string   `json:"route,omitempty"`
	Lang     string   `json:"lang,omitempty"`
}

type generateFeaturedImageOutput struct {
	Status             string   `json:"status"`
	Filename           string   `json:"filename"`
	PublicURL          string   `json:"public_url"`
	SizeKB             int      `json:"size_kb"`
	Style              string   `json:"style"`
	FrontmatterUpdated bool     `json:"frontmatter_updated"`
	LangsUpdated       []string `json:"langs_updated"`
	Deploy             string   `json:"deploy"`
}

var featuredImageSlugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
var featuredImageAccentRE = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func checkSriVersions(ctx context.Context, deps Deps, in checkSriVersionsInput) (checkSriVersionsOutput, error) {
	if err := ctx.Err(); err != nil {
		return checkSriVersionsOutput{}, err
	}
	if deps.Sri == nil {
		dryRun := boolValueDefault(in.DryRun, true)
		return checkSriVersionsOutput{
			Plugin:           "sri-check",
			Success:          false,
			ExitCode:         2,
			AutoFixRequested: boolValue(in.AutoFix),
			DryRun:           dryRun,
			Report: sri.Report{
				Exit:    2,
				Summary: "DISABLED",
				AutoFix: sri.AutoFixReport{
					Skipped: true,
				},
				Incident: sri.IncidentReport{Resolved: []string{}},
				DryRun:   dryRun,
			},
		}, nil
	}
	return deps.Sri.Check(ctx, sri.Request{
		AutoFix: in.AutoFix,
		DryRun:  in.DryRun,
	})
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

func generateFeaturedImage(ctx context.Context, deps Deps, in generateFeaturedImageInput) (generateFeaturedImageOutput, error) {
	if err := ctx.Err(); err != nil {
		return generateFeaturedImageOutput{}, err
	}
	if deps.Pages == nil {
		return generateFeaturedImageOutput{}, fmt.Errorf("pages service not configured")
	}
	if deps.PageMutations == nil {
		return generateFeaturedImageOutput{}, fmt.Errorf("page mutation service not configured")
	}

	style := strings.TrimSpace(in.Style)
	if style == "" {
		style = "tech"
	}
	if style != "tech" && style != "geo" {
		return generateFeaturedImageOutput{}, fmt.Errorf("style must be 'tech' or 'geo', got %q", style)
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return generateFeaturedImageOutput{}, fmt.Errorf("missing required field: title")
	}
	if utf8Len(title) > 80 {
		return generateFeaturedImageOutput{}, fmt.Errorf("title too long: max 80 chars")
	}
	subtitle := strings.TrimSpace(in.Subtitle)
	if utf8Len(subtitle) > 120 {
		return generateFeaturedImageOutput{}, fmt.Errorf("subtitle too long: max 120 chars")
	}
	if len(in.Tags) > 50 {
		return generateFeaturedImageOutput{}, fmt.Errorf("too many tags (max 50)")
	}
	for _, tag := range in.Tags {
		if utf8Len(tag) > 100 {
			return generateFeaturedImageOutput{}, fmt.Errorf("each tag must be a string ≤ 100 chars")
		}
	}
	slug := strings.TrimSpace(in.Slug)
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return generateFeaturedImageOutput{}, fmt.Errorf("slug must not contain path separators or ..")
	}
	if !featuredImageSlugRE.MatchString(slug) {
		return generateFeaturedImageOutput{}, fmt.Errorf("slug must be lowercase alphanumeric + hyphens")
	}
	accent := strings.TrimSpace(in.Accent)
	if accent != "" && !featuredImageAccentRE.MatchString(accent) {
		return generateFeaturedImageOutput{}, fmt.Errorf("accent must be a 6-digit hex color like #7aa2f7, got %q", accent)
	}
	if accent == "" {
		if style == "geo" {
			accent = "#bb9af7"
		} else {
			accent = "#7aa2f7"
		}
	}

	route := strings.TrimSpace(in.Route)
	langs, err := featuredImageTargetLangs(ctx, deps, route)
	if err != nil {
		return generateFeaturedImageOutput{}, err
	}

	filename := slug + "-featured.jpg"
	publicURL := "/images/" + filename
	stagingWorkspace := stagingFromDeps(deps)
	if stagingWorkspace == nil {
		return generateFeaturedImageOutput{}, fmt.Errorf("missing staging workspace")
	}
	outPath, err := safeFeaturedImagePath(stagingWorkspace, filename)
	if err != nil {
		return generateFeaturedImageOutput{}, err
	}

	result := generateFeaturedImageOutput{
		Status:    "ok",
		Filename:  filename,
		PublicURL: publicURL,
		Style:     style,
	}
	var cleanupPath string
	defer func() {
		if cleanupPath != "" && err != nil {
			_ = os.Remove(cleanupPath)
		}
	}()

	if err = os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return generateFeaturedImageOutput{}, err
	}
	if err = renderFeaturedImage(outPath, style, title, subtitle, in.Tags, accent); err != nil {
		return generateFeaturedImageOutput{}, err
	}
	cleanupPath = outPath
	info, err := os.Stat(outPath)
	if err != nil {
		return generateFeaturedImageOutput{}, err
	}
	result.SizeKB = int(info.Size() / 1024)

	if len(langs) > 0 {
		result.FrontmatterUpdated = true
		for _, lang := range langs {
			update := mutations.UpdatePageRequest{
				Route: route,
				Lang:  lang,
				Frontmatter: map[string]any{
					"featuredImage": publicURL,
				},
			}
			pageResult, updateErr := deps.PageMutations.Update(ctx, update)
			if updateErr != nil {
				return generateFeaturedImageOutput{}, updateErr
			}
			result.LangsUpdated = append(result.LangsUpdated, langOrDefault(lang))
			result.Deploy = pageResult.Deploy
		}
		return result, nil
	}

	if deps.Build == nil {
		return generateFeaturedImageOutput{}, fmt.Errorf("build service not configured")
	}
	buildRes, err := deps.Build.Build(ctx, mutations.BuildRequest{PurgeCF: false})
	if err != nil {
		return generateFeaturedImageOutput{}, err
	}
	result.Deploy = buildRes.Deploy
	return result, nil
}

func featuredImageTargetLangs(ctx context.Context, deps Deps, route string) ([]string, error) {
	if strings.TrimSpace(route) == "" {
		return nil, nil
	}
	langs := make([]string, 0, 3)
	for _, lang := range []string{"fr", "en"} {
		page, err := deps.Pages.Get(ctx, pages.GetRequest{Route: route, Lang: lang})
		if err == nil && strings.HasSuffix(page.File, "index."+lang+".md") {
			langs = append(langs, lang)
		}
	}
	if len(langs) > 0 {
		return langs, nil
	}
	if page, err := deps.Pages.Get(ctx, pages.GetRequest{Route: route}); err == nil && strings.HasSuffix(page.File, "index.md") {
		return []string{""}, nil
	}
	return nil, fmt.Errorf("Page not found: %s", route)
}

func stagingFromDeps(deps Deps) *staging.Workspace {
	switch {
	case deps.PageMutations != nil:
		return deps.PageMutations.Stage
	case deps.Build != nil:
		return deps.Build.Stage
	case deps.Pages != nil:
		return nil
	default:
		return nil
	}
}

func safeFeaturedImagePath(ws *staging.Workspace, filename string) (string, error) {
	if ws == nil {
		return "", fmt.Errorf("missing staging workspace")
	}
	rel := filepath.ToSlash(filepath.Join("images", filename))
	if existing, err := ws.ResolveExistingStatic(rel); err == nil {
		return existing, nil
	}
	return ws.ResolveNewStatic(rel)
}

func renderFeaturedImage(path string, style, title, subtitle string, tags []string, accent string) error {
	const (
		width  = 1200
		height = 675
	)
	bg1, bg2 := featuredImagePalette(style, title)
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height-1)
		row := blendColor(bg1, bg2, ratio)
		for x := 0; x < width; x++ {
			canvas.SetRGBA(x, y, row)
		}
	}

	accentRGBA := mustHexColor(accent)
	drawRect(canvas, 0, 0, 8, height, accentRGBA)
	drawRect(canvas, 8, height-6, width-8, 6, withAlpha(accentRGBA, 110))
	drawCircle(canvas, 72, 54, 16, withAlpha(accentRGBA, 45))
	drawCircle(canvas, 72, 54, 5, accentRGBA)

	drawText(canvas, 96, 60, "hugo-mcp-go", accentRGBA)
	drawTitle(canvas, 60, 438, title, accentRGBA)
	if subtitle != "" {
		drawWrappedText(canvas, 60, 500, subtitle, color.RGBA{235, 235, 235, 255}, 980)
	}
	for i, tag := range tags {
		if i >= 6 {
			break
		}
		x := 60 + i*178
		drawRoundedRect(canvas, x, 610, 160, 28, color.RGBA{0, 0, 0, 140}, withAlpha(accentRGBA, 200))
		drawCenteredText(canvas, x, 617, 160, "#"+tag, accentRGBA)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, canvas, &jpeg.Options{Quality: 88}); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func featuredImagePalette(style, title string) (color.RGBA, color.RGBA) {
	sum := sha1.Sum([]byte(style + "::" + title))
	base := colorFromHex(map[string]string{
		"geo":  "#2a2254",
		"tech": "#14243f",
	}[style])
	if base == (color.RGBA{}) {
		base = colorFromHex("#1a1b26")
	}
	shift := func(v byte, delta int) byte {
		n := int(v) + delta
		switch {
		case n < 0:
			return 0
		case n > 255:
			return 255
		default:
			return byte(n)
		}
	}
	variant := color.RGBA{
		R: shift(base.R, int(sum[0]%24)-12),
		G: shift(base.G, int(sum[1]%18)-9),
		B: shift(base.B, int(sum[2]%20)-10),
		A: 255,
	}
	return base, variant
}

func drawTitle(img *image.RGBA, x, y int, title string, accent color.RGBA) {
	drawWrappedText(img, x, y, title, color.RGBA{255, 255, 255, 255}, 1040)
	underlineY := y + 20
	drawRect(img, x, underlineY, 64, 4, accent)
}

func drawText(img *image.RGBA, x, y int, text string, clr color.RGBA) {
	drawString(img, x, y, text, clr, basicfont.Face7x13)
}

func drawCenteredText(img *image.RGBA, x, y, width int, text string, clr color.RGBA) {
	face := basicfont.Face7x13
	d := &font.Drawer{Dst: img, Src: image.NewUniform(clr), Face: face}
	textWidth := d.MeasureString(text).Round()
	startX := x + (width-textWidth)/2
	if startX < x+4 {
		startX = x + 4
	}
	drawString(img, startX, y, text, clr, face)
}

func drawWrappedText(img *image.RGBA, x, y int, text string, clr color.RGBA, maxWidth int) {
	lines := wrapText(text, basicfont.Face7x13, maxWidth)
	for i, line := range lines {
		drawString(img, x, y+i*18, line, clr, basicfont.Face7x13)
	}
}

func drawString(img *image.RGBA, x, y int, text string, clr color.RGBA, face font.Face) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(clr),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func wrapText(text string, face font.Face, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, 4)
	current := words[0]
	measure := func(s string) int {
		d := &font.Drawer{Face: face}
		return d.MeasureString(s).Round()
	}
	for _, word := range words[1:] {
		candidate := current + " " + word
		if measure(candidate) > maxWidth && current != "" {
			lines = append(lines, current)
			current = word
			continue
		}
		current = candidate
	}
	lines = append(lines, current)
	if len(lines) == 1 && measure(lines[0]) > maxWidth {
		runes := []rune(lines[0])
		mid := len(runes) / 2
		lines = []string{strings.TrimSpace(string(runes[:mid])), strings.TrimSpace(string(runes[mid:]))}
	}
	return lines
}

func drawRect(img *image.RGBA, x, y, w, h int, clr color.RGBA) {
	r := image.Rect(x, y, x+w, y+h)
	draw.Draw(img, r, image.NewUniform(clr), image.Point{}, draw.Src)
}

func drawRoundedRect(img *image.RGBA, x, y, w, h int, fill color.RGBA, stroke color.RGBA) {
	drawRect(img, x, y, w, h, fill)
	drawRect(img, x, y, w, 1, stroke)
	drawRect(img, x, y+h-1, w, 1, stroke)
	drawRect(img, x, y, 1, h, stroke)
	drawRect(img, x+w-1, y, 1, h, stroke)
}

func drawCircle(img *image.RGBA, cx, cy, radius int, clr color.RGBA) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, clr)
			}
		}
	}
}

func blendColor(a, b color.RGBA, ratio float64) color.RGBA {
	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(math.Round(v))
	}
	return color.RGBA{
		R: clamp(float64(a.R) + (float64(b.R)-float64(a.R))*ratio),
		G: clamp(float64(a.G) + (float64(b.G)-float64(a.G))*ratio),
		B: clamp(float64(a.B) + (float64(b.B)-float64(a.B))*ratio),
		A: 255,
	}
}

func colorFromHex(spec string) color.RGBA {
	var c color.RGBA
	if len(spec) != 7 || spec[0] != '#' {
		return c
	}
	raw, err := hexToBytes(spec[1:])
	if err != nil {
		return c
	}
	return color.RGBA{R: raw[0], G: raw[1], B: raw[2], A: 255}
}

func mustHexColor(hex string) color.RGBA {
	c := colorFromHex(hex)
	if c == (color.RGBA{}) {
		return color.RGBA{R: 122, G: 162, B: 247, A: 255}
	}
	return c
}

func withAlpha(c color.RGBA, alpha uint8) color.RGBA {
	c.A = alpha
	return c
}

func hexToBytes(s string) ([3]byte, error) {
	var out [3]byte
	if len(s) != 6 {
		return out, fmt.Errorf("invalid hex color")
	}
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 3 {
		return out, fmt.Errorf("invalid hex color")
	}
	copy(out[:], b)
	return out, nil
}

func utf8Len(s string) int {
	return len([]rune(strings.TrimSpace(s)))
}

func langOrDefault(lang string) string {
	if strings.TrimSpace(lang) == "" {
		return "default"
	}
	return lang
}
