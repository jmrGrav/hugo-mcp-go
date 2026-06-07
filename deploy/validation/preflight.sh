#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'preflight: %s\n' "$1" >&2
  exit 1
}

need_env() {
  local name="$1"
  local value="${!name:-}"
  [[ -n "$value" ]] || fail "missing required env $name"
}

canonical_path() {
  local path="$1"
  realpath "$path"
}

check_not_symlink() {
  local path="$1"
  [[ -e "$path" ]] || fail "missing path: $path"
  [[ ! -L "$path" ]] || fail "symlink not allowed: $path"
}

check_root() {
  local name="$1"
  local root
  root="${!name}"
  check_not_symlink "$root"
  local canonical
  canonical="$(canonical_path "$root")"
  [[ -d "$canonical" ]] || fail "not a directory: $root"
  printf '%s\n' "$canonical"
}

is_within_root() {
  local root="$1"
  local candidate="$2"
  [[ "$candidate" == "$root" || "$candidate" == "$root"/* ]]
}

check_mode_sane() {
  local path="$1"
  local mode
  mode="$(stat -c '%a' "$path")"
  local o
  o=$(( 8#$mode ))
  (( (o & 8#022) == 0 )) || fail "path is group/other writable: $path ($mode)"
}

need_env HUGO_ROOT
need_env HUGO_CONTENT_ROOT
need_env HUGO_STATIC_ROOT
need_env HUGO_SERVICE_USER
need_env HUGO_SERVICE_GROUP

HUGO_ROOT_CANON="$(check_root HUGO_ROOT)"
HUGO_CONTENT_ROOT_CANON="$(check_root HUGO_CONTENT_ROOT)"
HUGO_STATIC_ROOT_CANON="$(check_root HUGO_STATIC_ROOT)"

is_within_root "$HUGO_ROOT_CANON" "$HUGO_CONTENT_ROOT_CANON" || fail "HUGO_CONTENT_ROOT is outside HUGO_ROOT"
is_within_root "$HUGO_ROOT_CANON" "$HUGO_STATIC_ROOT_CANON" || fail "HUGO_STATIC_ROOT is outside HUGO_ROOT"

check_mode_sane "$HUGO_ROOT_CANON"
check_mode_sane "$HUGO_CONTENT_ROOT_CANON"
check_mode_sane "$HUGO_STATIC_ROOT_CANON"

current_user="$(id -un)"
current_group="$(id -gn)"
[[ "$current_user" == "$HUGO_SERVICE_USER" ]] || fail "unexpected user: $current_user"
[[ "$current_group" == "$HUGO_SERVICE_GROUP" ]] || fail "unexpected group: $current_group"

HUGO_BIN_EXPECTED="${HUGO_EXPECTED_HUGO_BIN:-}"
if [[ -z "$HUGO_BIN_EXPECTED" ]]; then
  fail "missing required env HUGO_EXPECTED_HUGO_BIN"
fi

check_not_symlink "$HUGO_BIN_EXPECTED"
[[ -x "$HUGO_BIN_EXPECTED" ]] || fail "Hugo binary not executable: $HUGO_BIN_EXPECTED"
HUGO_BIN_RESOLVED="$(command -v hugo || true)"
[[ -n "$HUGO_BIN_RESOLVED" ]] || fail "hugo not found on PATH"
HUGO_BIN_RESOLVED="$(canonical_path "$HUGO_BIN_RESOLVED")"
HUGO_BIN_EXPECTED_CANON="$(canonical_path "$HUGO_BIN_EXPECTED")"
[[ "$HUGO_BIN_RESOLVED" == "$HUGO_BIN_EXPECTED_CANON" ]] || fail "hugo resolves to $HUGO_BIN_RESOLVED, expected $HUGO_BIN_EXPECTED_CANON"
check_mode_sane "$HUGO_BIN_EXPECTED_CANON"

version_output="$(hugo version 2>/dev/null || true)"
[[ -n "$version_output" ]] || fail "unable to read hugo version"
if [[ -n "${HUGO_EXPECTED_HUGO_VERSION_PREFIX:-}" ]]; then
  [[ "$version_output" == "${HUGO_EXPECTED_HUGO_VERSION_PREFIX}"* ]] || fail "hugo version mismatch: $version_output"
fi

for p in "$HUGO_ROOT_CANON" "$HUGO_CONTENT_ROOT_CANON" "$HUGO_STATIC_ROOT_CANON"; do
  [[ ! -L "$p" ]] || fail "symlink not allowed: $p"
done

printf 'preflight: ok\n'
