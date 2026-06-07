package tools

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/pages"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/staging"
	"golang.org/x/image/font/basicfont"
)

func TestParityHelperBranches(t *testing.T) {
	t.Run("check sri canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := checkSriVersions(ctx, Deps{}, checkSriVersionsInput{}); err == nil {
			t.Fatal("checkSriVersions() expected context error")
		}
	})

	t.Run("stagingFromDeps picks page mutations", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := Deps{PageMutations: &mutations.PageService{Stage: ws}, Build: &mutations.BuildService{Stage: ws}}
		if got := stagingFromDeps(deps); got != ws {
			t.Fatalf("stagingFromDeps() = %#v want %#v", got, ws)
		}
	})

	t.Run("stagingFromDeps picks build", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := Deps{Build: &mutations.BuildService{Stage: ws}}
		if got := stagingFromDeps(deps); got != ws {
			t.Fatalf("stagingFromDeps() = %#v want %#v", got, ws)
		}
	})

	t.Run("stagingFromDeps nil", func(t *testing.T) {
		if got := stagingFromDeps(Deps{}); got != nil {
			t.Fatalf("stagingFromDeps() = %#v want nil", got)
		}
	})

	t.Run("feature image lang discovery", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{})

		// Create a bilingual leaf bundle to exercise the fr/en branch.
		if _, err := deps.PageMutations.Create(context.Background(), mutations.CreatePageRequest{
			Route:   "/posts/dual",
			Lang:    "fr",
			Title:   "Dual",
			Content: "fr body",
		}); err != nil {
			t.Fatalf("Create(fr) error = %v", err)
		}
		if _, err := deps.PageMutations.Create(context.Background(), mutations.CreatePageRequest{
			Route:   "/posts/dual",
			Lang:    "en",
			Title:   "Dual",
			Content: "en body",
		}); err != nil {
			t.Fatalf("Create(en) error = %v", err)
		}

		out, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Style:    "tech",
			Title:    "Dual",
			Slug:     "dual-feature",
			Route:    "/posts/dual",
			Subtitle: "bilingual",
			Tags:     []string{"fr", "en"},
		})
		if err != nil {
			t.Fatalf("generateFeaturedImage() error = %v", err)
		}
		if out.FrontmatterUpdated != true {
			t.Fatal("generateFeaturedImage() did not update frontmatter")
		}
		if len(out.LangsUpdated) != 2 {
			t.Fatalf("LangsUpdated = %#v want 2 entries", out.LangsUpdated)
		}

		// Second run should reuse the existing featured-image path branch.
		if _, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Style: "tech",
			Title: "Dual",
			Slug:  "dual-feature",
			Route: "/posts/dual",
		}); err != nil {
			t.Fatalf("second generateFeaturedImage() error = %v", err)
		}

		if _, err := os.Stat(filepath.Join(ws.StaticRoot, "images", "dual-feature-featured.jpg")); err != nil {
			t.Fatalf("generated image missing: %v", err)
		}
	})

	t.Run("featured image default page fallback", func(t *testing.T) {
		ws := newTestWorkspace(t)
		defaultPage := filepath.Join(ws.ContentRoot, "posts", "default")
		if err := os.MkdirAll(defaultPage, 0o755); err != nil {
			t.Fatalf("mkdir default page: %v", err)
		}
		raw := []byte("---\ntitle: Default\n---\n\nbody\n")
		if err := os.WriteFile(filepath.Join(defaultPage, "index.md"), raw, 0o644); err != nil {
			t.Fatalf("write default page: %v", err)
		}

		deps := newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{})
		out, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Style: "geo",
			Title: "Default",
			Slug:  "default-feature",
			Route: "/posts/default",
		})
		if err != nil {
			t.Fatalf("generateFeaturedImage() error = %v", err)
		}
		if len(out.LangsUpdated) != 1 || out.LangsUpdated[0] != "default" {
			t.Fatalf("LangsUpdated = %#v want default", out.LangsUpdated)
		}
	})

	t.Run("featured image missing build when route absent", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := Deps{
			Pages:         pages.New(ws.ContentRoot),
			PageMutations: mutations.NewPageService(ws),
		}
		if _, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Style: "tech",
			Title: "No Build",
			Slug:  "no-build",
		}); err == nil || !strings.Contains(err.Error(), "build service not configured") {
			t.Fatalf("generateFeaturedImage() error = %v want build service not configured", err)
		}
	})

	t.Run("featured image missing pages service", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := Deps{
			PageMutations: mutations.NewPageService(ws),
		}
		if _, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Style: "tech",
			Title: "No Pages",
			Slug:  "no-pages",
		}); err == nil || !strings.Contains(err.Error(), "pages service not configured") {
			t.Fatalf("generateFeaturedImage() error = %v want pages service not configured", err)
		}
	})

	t.Run("featured image missing page mutations service", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := Deps{
			Pages: pages.New(ws.ContentRoot),
		}
		if _, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Style: "tech",
			Title: "No Mutations",
			Slug:  "no-mutations",
		}); err == nil || !strings.Contains(err.Error(), "page mutation service not configured") {
			t.Fatalf("generateFeaturedImage() error = %v want page mutation service not configured", err)
		}
	})

	t.Run("featured image no route build fallback", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{})
		out, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Style: "tech",
			Title: "Build Fallback",
			Slug:  "build-fallback",
		})
		if err != nil {
			t.Fatalf("generateFeaturedImage() error = %v", err)
		}
		if out.Deploy == "" {
			t.Fatal("generateFeaturedImage() did not return deploy output")
		}
	})

	t.Run("featured image defaults", func(t *testing.T) {
		ws := newTestWorkspace(t)
		deps := newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{})
		out, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
			Title: "Defaults",
			Slug:  "defaults",
		})
		if err != nil {
			t.Fatalf("generateFeaturedImage() error = %v", err)
		}
		if out.Style != "tech" {
			t.Fatalf("Style = %q want tech", out.Style)
		}
	})

	t.Run("featured image helpers", func(t *testing.T) {
		if got := langOrDefault(""); got != "default" {
			t.Fatalf("langOrDefault(\"\") = %q want default", got)
		}
		if got := langOrDefault("fr"); got != "fr" {
			t.Fatalf("langOrDefault(fr) = %q want fr", got)
		}
		if got := utf8Len("héllo"); got != 5 {
			t.Fatalf("utf8Len() = %d want 5", got)
		}
		if _, err := hexToBytes("ff00"); err == nil {
			t.Fatal("hexToBytes() expected length error")
		}
		if _, err := hexToBytes("zzzzzz"); err == nil {
			t.Fatal("hexToBytes() expected decode error")
		}
		if c := colorFromHex("#112233"); c.R != 0x11 || c.G != 0x22 || c.B != 0x33 {
			t.Fatalf("colorFromHex() = %#v", c)
		}
		if c := mustHexColor("bad"); c == (color.RGBA{}) {
			t.Fatal("mustHexColor() returned zero color")
		}
		if got := blendColor(color.RGBA{R: 0, G: 0, B: 0, A: 255}, color.RGBA{R: 255, G: 128, B: 64, A: 255}, 0.5); got.R == 0 || got.G == 0 {
			t.Fatalf("blendColor() = %#v", got)
		}
		if got := blendColor(color.RGBA{R: 10, G: 20, B: 30, A: 255}, color.RGBA{R: 200, G: 210, B: 220, A: 255}, -1); got.R != 0 || got.G != 0 {
			t.Fatalf("blendColor(lower clamp) = %#v", got)
		}
		if got := blendColor(color.RGBA{R: 10, G: 20, B: 30, A: 255}, color.RGBA{R: 200, G: 210, B: 220, A: 255}, 2); got.R != 255 || got.G != 255 {
			t.Fatalf("blendColor(upper clamp) = %#v", got)
		}
		if a, b := featuredImagePalette("tech", "Alpha"); a == (color.RGBA{}) || b == (color.RGBA{}) {
			t.Fatal("featuredImagePalette() returned zero colors")
		}
		if a, b := featuredImagePalette("geo", "Alpha"); a == (color.RGBA{}) || b == (color.RGBA{}) {
			t.Fatal("featuredImagePalette(geo) returned zero colors")
		}
		if a, b := featuredImagePalette("unknown", "Alpha"); a == (color.RGBA{}) || b == (color.RGBA{}) {
			t.Fatal("featuredImagePalette(unknown) returned zero colors")
		}
		lines := wrapText("one two three four five six seven eight nine", basicfont.Face7x13, 40)
		if len(lines) < 2 {
			t.Fatalf("wrapText() = %#v want multiple lines", lines)
		}
		if got := wrapText("", basicfont.Face7x13, 40); got != nil {
			t.Fatalf("wrapText(empty) = %#v want nil", got)
		}
		if got := wrapText("short", basicfont.Face7x13, 500); len(got) != 1 {
			t.Fatalf("wrapText(short) = %#v want one line", got)
		}
		if got := wrapText("supercalifragilisticexpialidocious", basicfont.Face7x13, 40); len(got) != 2 {
			t.Fatalf("wrapText(long word) = %#v want 2 lines", got)
		}
		canvas := image.NewRGBA(image.Rect(0, 0, 40, 40))
		drawCenteredText(canvas, 0, 10, 5, "center me", color.RGBA{255, 255, 255, 255})
	})
}

func TestGenerateFeaturedImageInvalidStyleAndContext(t *testing.T) {
	ws := newTestWorkspace(t)
	deps := newMutationDepsWithWorkspace(t, ws, &fakeBuildRunner{})
	_, err := generateFeaturedImage(context.Background(), deps, generateFeaturedImageInput{
		Style: "bad",
		Title: "Bad",
		Slug:  "bad",
	})
	if err == nil {
		t.Fatal("generateFeaturedImage() expected invalid style error")
	}

	for _, tc := range []generateFeaturedImageInput{
		{Style: "tech", Slug: "bad"},
		{Style: "tech", Title: strings.Repeat("a", 81), Slug: "bad"},
		{Style: "tech", Title: "Bad", Subtitle: strings.Repeat("b", 121), Slug: "bad"},
		{Style: "tech", Title: "Bad", Slug: "../escape"},
		{Style: "tech", Title: "Bad", Slug: "bad", Tags: make([]string, 51)},
	} {
		if _, err := generateFeaturedImage(context.Background(), deps, tc); err == nil {
			t.Fatalf("generateFeaturedImage(%#v) expected validation error", tc)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	cancel()
	if _, err := generateFeaturedImage(ctx, deps, generateFeaturedImageInput{
		Style: "tech",
		Title: "Bad",
		Slug:  "bad",
	}); err == nil {
		t.Fatal("generateFeaturedImage() expected context error")
	}
}

func TestFeaturedImagePathResolutionBranches(t *testing.T) {
	ws := newTestWorkspace(t)
	path, err := safeFeaturedImagePath(ws, "example-featured.jpg")
	if err != nil {
		t.Fatalf("safeFeaturedImagePath() error = %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("static", "images", "example-featured.jpg")) && !strings.HasSuffix(path, "/static/images/example-featured.jpg") {
		t.Fatalf("safeFeaturedImagePath() = %q", path)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write featured image: %v", err)
	}
	path2, err := safeFeaturedImagePath(ws, "example-featured.jpg")
	if err != nil {
		t.Fatalf("safeFeaturedImagePath(existing) error = %v", err)
	}
	if path2 != path {
		t.Fatalf("safeFeaturedImagePath(existing) = %q want %q", path2, path)
	}

	bad := &staging.Workspace{HugoRoot: t.TempDir(), StaticRoot: filepath.Join(t.TempDir(), "missing")}
	if _, err := safeFeaturedImagePath(bad, "bad.jpg"); err == nil {
		t.Fatal("safeFeaturedImagePath() expected error for invalid workspace")
	}
}
