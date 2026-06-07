#!/usr/bin/env bash
set -euo pipefail

URL="${MCP_URL:-http://127.0.0.1:18181/mcp}"
TOKEN="${MCP_TOKEN:-local-token}"
CURL_BIN="${CURL_BIN:-curl}"

req() {
  local body="$1"
  "$CURL_BIN" -skS \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    --data "$body" \
    "$URL"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'expected to find %s in %s\n' "$needle" "$haystack" >&2
    exit 1
  fi
}

init_body='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{"roots":{"listChanged":true}},"clientInfo":{"name":"parity-smoke","version":"0.1.0"}}}'
tools_pre_body='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
tools_null_body='{"jsonrpc":"2.0","id":null,"method":"tools/list","params":{}}'
resources_body='{"jsonrpc":"2.0","id":3,"method":"resources/list","params":{}}'
prompts_body='{"jsonrpc":"2.0","id":4,"method":"prompts/list","params":{}}'
unknown_body='{"jsonrpc":"2.0","id":5,"method":"does/not_exist","params":{}}'
notif_noid_body='{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'
notif_id_body='{"jsonrpc":"2.0","id":6,"method":"notifications/initialized","params":{}}'

tools_pre_resp="$(req "$tools_pre_body")"
assert_contains "$tools_pre_resp" '"id":2'
assert_contains "$tools_pre_resp" '"tools"'

init_resp="$(req "$init_body")"
assert_contains "$init_resp" '"protocolVersion":"2025-03-26"'
assert_contains "$init_resp" '"serverInfo":{"name":"hugo-mcp","version":"1.0.0"}'

tools_null_resp="$(req "$tools_null_body")"
assert_contains "$tools_null_resp" '"id":null'
assert_contains "$tools_null_resp" '"tools"'

resources_resp="$(req "$resources_body")"
assert_contains "$resources_resp" '"code":-32601'
assert_contains "$resources_resp" 'Method not found: resources/list'

prompts_resp="$(req "$prompts_body")"
assert_contains "$prompts_resp" '"code":-32601'
assert_contains "$prompts_resp" 'Method not found: prompts/list'

unknown_resp="$(req "$unknown_body")"
assert_contains "$unknown_resp" '"code":-32601'
assert_contains "$unknown_resp" 'Method not found: does/not_exist'

notif_noid_resp="$(req "$notif_noid_body")"
if [[ -n "$notif_noid_resp" ]]; then
  printf 'expected empty response for notifications/initialized without id, got: %s\n' "$notif_noid_resp" >&2
  exit 1
fi

notif_id_resp="$(req "$notif_id_body")"
assert_contains "$notif_id_resp" '"code":-32600'
assert_contains "$notif_id_resp" 'unexpected id for'

printf 'parity-smoke ok\n'
