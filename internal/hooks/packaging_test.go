package hooks

import (
	"os"
	"strings"
	"testing"
)

func TestPackagingDocsAndEnvExampleStayNonSecret(t *testing.T) {
	envExample := mustReadFile(t, "../../deploy/systemd/hugo-mcp-go.env.example")
	for _, want := range []string{
		"HUGO_HOOKS_DRY_RUN=true",
		"HUGO_CLOUDFLARE_TOKEN_FILE=/etc/hugo-mcp-go/secrets/cloudflare_api_token",
		"HUGO_GOOGLE_INDEXING_SERVICE_ACCOUNT_FILE=/etc/hugo-mcp-go/secrets/google_indexing_service_account.json",
		"HUGO_INDEXNOW_KEY_FILE=/etc/hugo-mcp-go/secrets/indexnow_key",
		"HUGO_HOOKS_DB=/var/lib/hugo-mcp-go/hooks.db",
	} {
		if !strings.Contains(envExample, want) {
			t.Fatalf("env example missing %q", want)
		}
	}
	for _, banned := range []string{"PRIVATE KEY", "Bearer ", "client_secret=", "access_token"} {
		if strings.Contains(envExample, banned) {
			t.Fatalf("env example must not contain %q", banned)
		}
	}

	postBuild := mustReadFile(t, "../../docs/POST_BUILD_HOOKS.md")
	if !strings.Contains(strings.ToLower(postBuild), "dry-run") {
		t.Fatal("POST_BUILD_HOOKS.md must mention dry-run")
	}
	if !strings.Contains(strings.ToLower(postBuild), "sqlite") {
		t.Fatal("POST_BUILD_HOOKS.md must mention sqlite")
	}
	if !strings.Contains(strings.ToLower(postBuild), "file-backed") {
		t.Fatal("POST_BUILD_HOOKS.md must mention file-backed secrets")
	}

	secrets := mustReadFile(t, "../../docs/SECRETS_MODEL.md")
	if !strings.Contains(strings.ToLower(secrets), "fail-closed") {
		t.Fatal("SECRETS_MODEL.md must mention fail-closed secret handling")
	}
	if !strings.Contains(strings.ToLower(secrets), "group hugo-mcp") {
		t.Fatal("SECRETS_MODEL.md must mention group hugo-mcp permissions")
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
