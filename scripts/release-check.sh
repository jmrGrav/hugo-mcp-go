#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
BUILD_SCRIPT="$ROOT/scripts/build-release.sh"
GOFMT_FILES=()
MIN_GOVERSION="go1.25.11"

export PATH="$HOME/go/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin"
export GOTOOLCHAIN="${RELEASE_GOTOOLCHAIN:-go1.25.11+auto}"

usage() {
  cat <<'EOF'
Usage: scripts/release-check.sh [--step test|race|vet|vuln|secrets|build|all]

Default behavior runs the full release validation pipeline in order.
EOF
}

require_tool() {
  local tool="$1"
  local install_hint="$2"
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "missing required tool: $tool" >&2
    echo "install: $install_hint" >&2
    exit 127
  fi
}

check_scanner_tools() {
  local missing=0
  for tool in govulncheck gitleaks trufflehog; do
    case "$tool" in
      govulncheck)
        if ! command -v "$tool" >/dev/null 2>&1; then
          echo "missing required tool: govulncheck" >&2
          echo "install: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
          missing=1
        fi
        ;;
      gitleaks)
        if ! command -v "$tool" >/dev/null 2>&1; then
          echo "missing required tool: gitleaks" >&2
          echo "install: go install github.com/gitleaks/gitleaks/v8@latest" >&2
          missing=1
        fi
        ;;
      trufflehog)
        if ! command -v "$tool" >/dev/null 2>&1; then
          echo "missing required tool: trufflehog" >&2
          echo "install: go install github.com/trufflesecurity/trufflehog/v3@latest" >&2
          missing=1
        fi
        ;;
    esac
  done
  if (( missing != 0 )); then
    exit 127
  fi
}

check_go_toolchain() {
  local current
  current=$(go env GOVERSION)
  if [[ "$(printf '%s\n%s\n' "$MIN_GOVERSION" "$current" | sort -V | head -n1)" != "$MIN_GOVERSION" ]]; then
    echo "Go toolchain too old: $current" >&2
    echo "minimum required: $MIN_GOVERSION" >&2
    echo "set RELEASE_GOTOOLCHAIN=go1.25.11+auto or install a newer Go toolchain" >&2
    exit 1
  fi
}

check_gofmt() {
  mapfile -t GOFMT_FILES < <(find "$ROOT" -type f -name '*.go' \
    -not -path "$ROOT/dist/*" \
    -not -path "$ROOT/.git/*" \
    -not -path "$ROOT/vendor/*" \
    -print | sort | xargs gofmt -l)
  if (( ${#GOFMT_FILES[@]} > 0 )); then
    printf 'gofmt needed for:\n' >&2
    printf '  %s\n' "${GOFMT_FILES[@]}" >&2
    echo "run: gofmt -w <files>" >&2
    exit 1
  fi
}

run_tests() {
  (cd "$ROOT" && go test ./...)
}

run_race() {
  (cd "$ROOT" && go test -race ./...)
}

run_vet() {
  (cd "$ROOT" && go vet ./...)
}

run_vuln() {
  require_tool govulncheck 'go install golang.org/x/vuln/cmd/govulncheck@latest'
  (cd "$ROOT" && govulncheck ./...)
}

run_gitleaks() {
  require_tool gitleaks 'go install github.com/gitleaks/gitleaks/v8@latest'
  (cd "$ROOT" && gitleaks detect --source "$ROOT" --no-banner --redact)
}

run_trufflehog() {
  require_tool trufflehog 'go install github.com/trufflesecurity/trufflehog/v3@latest'
  (cd "$ROOT" && trufflehog filesystem --directory "$ROOT" --no-update --no-verification --fail --fail-on-scan-errors --force-skip-binaries --force-skip-archives)
}

run_build_local() {
  "$BUILD_SCRIPT" local
}

run_build_multi() {
  PRESERVE_DIST=1 "$BUILD_SCRIPT" multi
}

run_checksums() {
  "$BUILD_SCRIPT" checksum
  (cd "$ROOT" && sha256sum -c dist/SHA256SUMS)
}

run_pipeline() {
  check_go_toolchain
  check_gofmt
  run_tests
  run_race
  run_vet
  check_scanner_tools
  run_vuln
  run_gitleaks
  run_trufflehog
  run_build_local
  run_build_multi
  run_checksums
}

case "${1:-}" in
  "")
    run_pipeline
    ;;
  --step)
    case "${2:-}" in
      test) run_tests ;;
      race) run_race ;;
      vet) run_vet ;;
      vuln) run_vuln ;;
      secrets)
        run_gitleaks
        run_trufflehog
        ;;
      build)
        run_build_local
        run_build_multi
        run_checksums
        ;;
      all) run_pipeline ;;
      *)
        usage >&2
        exit 2
        ;;
    esac
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
