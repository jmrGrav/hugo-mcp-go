package frontmatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content", "posts", "bonjour", "index.fr.md")
	fm, content, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if content != "Contenu français minimal." {
		t.Fatalf("ParseFile() content = %q", content)
	}
	if got := fm["title"]; got != "Bonjour" {
		t.Fatalf("ParseFile() title = %#v", got)
	}
	tags, ok := fm["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("ParseFile() tags = %#v", fm["tags"])
	}
}

func TestSplitNoFrontmatter(t *testing.T) {
	fm, content, err := Split([]byte("plain content\n"))
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(fm) != 0 || content != "plain content\n" {
		t.Fatalf("Split() = %#v %q", fm, content)
	}
}

func TestRenderRoundTrip(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content", "_index.fr.md"))
	if err != nil {
		t.Fatal(err)
	}
	fm, content, err := Split(raw)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := Render(fm, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered) == 0 {
		t.Fatal("Render() returned empty output")
	}
}

func TestParseUnicodeFrontmatter(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "testdata", "fixtures", "minimal-site", "content", "posts", "unicode", "index.fr.md")
	fm, content, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if got := fm["title"]; got != "Café déjà vu" {
		t.Fatalf("ParseFile() title = %#v", got)
	}
	if !strings.Contains(content, "🌍") {
		t.Fatalf("ParseFile() unicode content missing: %q", content)
	}
}

func TestParseLargeMarkdown(t *testing.T) {
	var body strings.Builder
	body.Grow(1 << 14)
	for i := 0; i < 512; i++ {
		body.WriteString("Large markdown line ")
		body.WriteString(strings.Repeat("x", 16))
		body.WriteByte('\n')
	}
	fm, content, err := Split([]byte("---\ntitle: Large\n---\n\n" + body.String()))
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if fm["title"] != "Large" {
		t.Fatalf("Split() title = %#v", fm["title"])
	}
	if len(content) < 10000 {
		t.Fatalf("Split() content too short: %d", len(content))
	}
}
