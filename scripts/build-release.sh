#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DIST="$ROOT/dist"
BINS=("hugo-mcp-go" "hugo-mcp-shim")
MULTI_PLATFORMS=("linux/amd64" "linux/arm64")
LOCAL_GOOS=$(go env GOOS)
LOCAL_GOARCH=$(go env GOARCH)
BUILD_FLAGS=(-trimpath -buildvcs=false -ldflags "-s -w")

usage() {
  cat <<'EOF'
Usage: scripts/build-release.sh [local|multi|all|checksum]

Environment:
  PRESERVE_DIST=1  Keep existing dist/ contents instead of deleting them first.
EOF
}

clean_dist() {
  if [[ "${PRESERVE_DIST:-0}" != "1" ]]; then
    rm -rf "$DIST"
  fi
  mkdir -p "$DIST"
}

build_binary() {
  local os="$1"
  local arch="$2"
  local prefix="$3"
  local bin="$4"
  local outdir="$DIST/${os}_${arch}"

  if [[ -n "$prefix" ]]; then
    outdir="$DIST/$prefix/${os}_${arch}"
  fi

  mkdir -p "$outdir"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build "${BUILD_FLAGS[@]}" -o "$outdir/$bin" "$ROOT/cmd/$bin"
}

build_local() {
  clean_dist
  for bin in "${BINS[@]}"; do
    build_binary "$LOCAL_GOOS" "$LOCAL_GOARCH" local "$bin"
  done
}

build_multi() {
  clean_dist
  for platform in "${MULTI_PLATFORMS[@]}"; do
    IFS=/ read -r os arch <<<"$platform"
    for bin in "${BINS[@]}"; do
      build_binary "$os" "$arch" "" "$bin"
    done
  done
}

build_all() {
  clean_dist
  for bin in "${BINS[@]}"; do
    build_binary "$LOCAL_GOOS" "$LOCAL_GOARCH" local "$bin"
  done
  for platform in "${MULTI_PLATFORMS[@]}"; do
    IFS=/ read -r os arch <<<"$platform"
    for bin in "${BINS[@]}"; do
      build_binary "$os" "$arch" "" "$bin"
    done
  done
  generate_checksums
}

generate_checksums() {
  mkdir -p "$DIST"
  (
    cd "$ROOT"
    find dist -type f \( -name 'hugo-mcp-go' -o -name 'hugo-mcp-shim' \) ! -name 'SHA256SUMS' -print0 \
      | sort -z \
      | xargs -0 sha256sum > dist/SHA256SUMS
  )
}

case "${1:-all}" in
  local)
    build_local
    ;;
  multi)
    build_multi
    ;;
  all)
    build_all
    ;;
  checksum)
    generate_checksums
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "Unknown build mode: $1" >&2
    usage >&2
    exit 2
    ;;
esac
