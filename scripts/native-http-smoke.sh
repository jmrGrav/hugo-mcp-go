#!/usr/bin/env bash
set -euo pipefail

CURL_BIN="${CURL_BIN:-curl}"
PYTHON_BIN="${PYTHON_BIN:-python3}"
URL="${MCP_URL:-http://127.0.0.1:18181/mcp}"

load_token() {
  local token="${1:-}"
  local token_file="${2:-}"
  if [[ -n "$token" ]]; then
    printf '%s' "$token"
    return 0
  fi
  if [[ -n "$token_file" && -r "$token_file" ]]; then
    tr -d '\r\n' < "$token_file"
    return 0
  fi
  return 1
}

call_rpc() {
  local body="$1"
  local resp_file status_file
  resp_file="$(mktemp)"
  status_file="$(mktemp)"
  "$CURL_BIN" -sS \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    --data "$body" \
    -o "$resp_file" \
    -w '%{http_code}' \
    "$URL" >"$status_file"
  printf '%s %s\n' "$(cat "$status_file")" "$resp_file"
}

json_path() {
  local file="$1"
  local path="$2"
  "$PYTHON_BIN" - "$file" "$path" <<'PY'
import json
import sys

raw = open(sys.argv[1], "r", encoding="utf-8").read().strip()
value = json.loads(raw)
for part in sys.argv[2].split("."):
    if part.isdigit():
        value = value[int(part)]
    else:
        value = value[part]
if isinstance(value, (dict, list)):
    print(json.dumps(value, separators=(",", ":")))
elif isinstance(value, bool):
    print(str(value).lower())
else:
    print(value)
PY
}

tool_text_json_path() {
  local file="$1"
  local path="$2"
  "$PYTHON_BIN" - "$file" "$path" <<'PY'
import json
import sys

raw = open(sys.argv[1], "r", encoding="utf-8").read().strip()
outer = json.loads(raw)
value = outer["result"]["content"][0]["text"]
decoded = json.loads(value)
for part in sys.argv[2].split("."):
    if part.isdigit():
        decoded = decoded[int(part)]
    else:
        decoded = decoded[part]
if isinstance(decoded, (dict, list)):
    print(json.dumps(decoded, separators=(",", ":")))
elif isinstance(decoded, bool):
    print(str(decoded).lower())
else:
    print(decoded)
PY
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'expected to find %s in %s\n' "$needle" "$haystack" >&2
    exit 1
  fi
}

assert_status() {
  local got="$1"
  local want="$2"
  local label="$3"
  if [[ "$got" != "$want" ]]; then
    printf '%s: status %s want %s\n' "$label" "$got" "$want" >&2
    exit 1
  fi
}

TOKEN="$(load_token "${MCP_TOKEN:-}" "${MCP_TOKEN_FILE:-}")" || {
  printf 'MCP_TOKEN or MCP_TOKEN_FILE is required\n' >&2
  exit 1
}

printf '[native-http-smoke] endpoint=%s\n' "$URL"

read -r status resp_file < <(call_rpc '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{"roots":{"listChanged":true}},"clientInfo":{"name":"native-http-smoke","version":"0.1.0"}}}')
assert_status "$status" 200 "initialize"
assert_contains "$(json_path "$resp_file" 'result.protocolVersion')" "2025-03-26"
assert_contains "$(json_path "$resp_file" 'result.serverInfo.name')" "hugo-mcp"

read -r status resp_file < <(call_rpc '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
assert_status "$status" 200 "tools/list"
for tool in build_site check_sri_versions generate_featured_image get_asset_chunk get_page_chunk list_assets list_pages; do
  assert_contains "$(json_path "$resp_file" 'result.tools')" "$tool"
done

read -r status resp_file < <(call_rpc '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_pages","arguments":{"limit":1}}}')
assert_status "$status" 200 "list_pages"
assert_contains "$(json_path "$resp_file" 'result.structuredContent.has_more')" "true"
page_cursor="$(json_path "$resp_file" 'result.structuredContent.next_cursor')"
page_route="$(json_path "$resp_file" 'result.structuredContent.pages.0.route')"
page_lang="$(json_path "$resp_file" 'result.structuredContent.pages.0.lang')"

read -r status resp_file < <(call_rpc "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/call\",\"params\":{\"name\":\"list_pages\",\"arguments\":{\"limit\":1,\"cursor\":\"${page_cursor}\"}}}")
assert_status "$status" 200 "list_pages cursor"
assert_contains "$(json_path "$resp_file" 'result.structuredContent.pages')" "["

read -r status resp_file < <(call_rpc "{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"tools/call\",\"params\":{\"name\":\"get_page\",\"arguments\":{\"route\":\"${page_route}\",\"lang\":\"${page_lang}\"}}}")
assert_status "$status" 200 "get_page"
assert_contains "$(json_path "$resp_file" 'result.structuredContent.route')" "/"

read -r status resp_file < <(call_rpc "{\"jsonrpc\":\"2.0\",\"id\":6,\"method\":\"tools/call\",\"params\":{\"name\":\"get_page_chunk\",\"arguments\":{\"route\":\"${page_route}\",\"lang\":\"${page_lang}\",\"cursor\":0,\"chunk_bytes\":32}}}")
assert_status "$status" 200 "get_page_chunk"
if [[ -z "$(json_path "$resp_file" 'result.structuredContent.content')" ]]; then
  printf 'get_page_chunk returned empty chunk\n' >&2
  exit 1
fi

read -r status resp_file < <(call_rpc '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"list_assets","arguments":{"max_results":1}}}')
assert_status "$status" 200 "list_assets"
assert_contains "$(json_path "$resp_file" 'result.structuredContent.assets')" "["
asset_path="$(json_path "$resp_file" 'result.structuredContent.assets.0.path')"

read -r status resp_file < <(call_rpc "{\"jsonrpc\":\"2.0\",\"id\":8,\"method\":\"tools/call\",\"params\":{\"name\":\"get_asset_chunk\",\"arguments\":{\"path\":\"${asset_path}\",\"cursor\":0,\"chunk_bytes\":8}}}")
assert_status "$status" 200 "get_asset_chunk"
if [[ -z "$(json_path "$resp_file" 'result.structuredContent.chunk')" ]]; then
  printf 'get_asset_chunk returned empty chunk\n' >&2
  exit 1
fi

read -r status resp_file < <(call_rpc '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"build_site","arguments":{"purge_cf":false}}}')
assert_status "$status" 200 "build_site"
assert_contains "$(json_path "$resp_file" 'result.structuredContent.status')" "built"

read -r status resp_file < <(call_rpc '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"check_sri_versions","arguments":{"auto_fix":false,"dry_run":true}}}')
assert_status "$status" 200 "check_sri_versions"
assert_contains "$(json_path "$resp_file" 'result.structuredContent.plugin')" "sri-check"
assert_contains "$(json_path "$resp_file" 'result.structuredContent.report.dry_run')" "true"

printf 'native-http-smoke ok\n'
