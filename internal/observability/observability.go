package observability

import (
	"log/slog"
	"os"
	"regexp"
)

var (
	absolutePathRe = regexp.MustCompile(`(?:/[^ \t\n\r"'` + "`" + `]+)+|(?:[A-Za-z]:\\\\[^ \t\n\r"'` + "`" + `]+)+`)
	bearerRe       = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+\b`)
)

func New() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func RedactString(s string) string {
	s = bearerRe.ReplaceAllString(s, "Bearer <redacted>")
	return absolutePathRe.ReplaceAllStringFunc(s, func(_ string) string {
		return "<redacted-path>"
	})
}
