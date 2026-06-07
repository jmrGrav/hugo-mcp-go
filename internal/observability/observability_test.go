package observability

import (
	"strings"
	"testing"
)

func TestRedactString(t *testing.T) {
	got := RedactString(`err="/home/jm/Documents/hugo-mcp-go/internal/config/config.go" token=Bearer abc123 C:\Users\jm\private.txt`)
	if got == "" {
		t.Fatal("RedactString() returned empty")
	}
	if got == `err="/home/jm/Documents/hugo-mcp-go/internal/config/config.go" token=Bearer abc123 C:\Users\jm\private.txt` {
		t.Fatal("RedactString() did not redact anything")
	}
	if !strings.Contains(got, "<redacted-path>") {
		t.Fatalf("RedactString() missing path redaction: %q", got)
	}
	if !strings.Contains(got, "Bearer <redacted>") {
		t.Fatalf("RedactString() missing bearer redaction: %q", got)
	}
}
