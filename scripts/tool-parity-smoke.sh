#!/usr/bin/env bash
set -euo pipefail

CURL_BIN="${CURL_BIN:-curl}"
PYTHON_BIN="${PYTHON_BIN:-python3}"

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
  local url="$1"
  local token="$2"
  local body="$3"
  local resp_file status_file
  resp_file="$(mktemp)"
  status_file="$(mktemp)"
  "$CURL_BIN" -sS \
    -H "Authorization: Bearer ${token}" \
    -H "Content-Type: application/json" \
    --data "$body" \
    -o "$resp_file" \
    -w '%{http_code}' \
    "$url" >"$status_file"
  printf '%s %s\n' "$(cat "$status_file")" "$resp_file"
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

json_path() {
  local file="$1"
  local path="$2"
  "$PYTHON_BIN" - "$file" "$path" <<'PY'
import json
import sys

path = sys.argv[2]
raw = open(sys.argv[1], "r", encoding="utf-8").read().strip()
if not raw:
    sys.exit(0)
value = json.loads(raw)
for part in path.split("."):
    if part.isdigit():
        value = value[int(part)]
    else:
        value = value[part]
if isinstance(value, (dict, list)):
    print(json.dumps(value, separators=(",", ":")))
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

file_path = sys.argv[1]
path = sys.argv[2]
raw = open(file_path, "r", encoding="utf-8").read().strip()
if not raw:
    sys.exit(1)
outer = json.loads(raw)
text = outer["result"]["content"][0]["text"]
value = json.loads(text)
for part in path.split("."):
    if part.isdigit():
        value = value[int(part)]
    else:
        value = value[part]
if isinstance(value, (dict, list)):
    print(json.dumps(value, separators=(",", ":")))
else:
    print(value)
PY
}

assert_json_error() {
  local file="$1"
  local code="$2"
  local want_substr="$3"
  "$PYTHON_BIN" - "$file" "$code" "$want_substr" <<'PY'
import json
import sys

file_path, want_code, want_sub = sys.argv[1], int(sys.argv[2]), sys.argv[3]
raw = open(file_path, "r", encoding="utf-8").read().strip()
data = json.loads(raw)
err = data.get("error")
if not err:
    raise SystemExit("missing error object")
if err.get("code") != want_code:
    raise SystemExit(f"error code {err.get('code')} want {want_code}")
if want_sub not in str(err.get("message", "")):
    raise SystemExit(f"error message {err.get('message')!r} missing {want_sub!r}")
PY
}

assert_tool_error() {
  local file="$1"
  local want_sub="$2"
  "$PYTHON_BIN" - "$file" "$want_sub" <<'PY'
import json
import sys

file_path, want_sub = sys.argv[1], sys.argv[2]
raw = open(file_path, "r", encoding="utf-8").read().strip()
data = json.loads(raw)
result = data.get("result", {})
if not result.get("isError"):
    raise SystemExit("tool did not return isError=true")
texts = [content.get("text", "") for content in result.get("content", []) if isinstance(content, dict)]
if not any(want_sub in text for text in texts):
    raise SystemExit(f"tool error text missing {want_sub!r}: {texts!r}")
PY
}

assert_result_list_or_error() {
  local file="$1"
  local field="$2"
  "$PYTHON_BIN" - "$file" "$field" <<'PY'
import json
import sys

file_path, field = sys.argv[1], sys.argv[2]
raw = open(file_path, "r", encoding="utf-8").read().strip()
data = json.loads(raw)
if "error" in data:
    err = data["error"]
    msg = str(err.get("message", ""))
    if err.get("code") not in (0, -32601) or ("unsupported" not in msg and "Method not found" not in msg):
        raise SystemExit(f"unexpected error response: {err!r}")
    sys.exit(0)
result = data.get("result", {})
value = result.get(field)
if value != []:
    raise SystemExit(f"result.{field} = {value!r} want []")
PY
}

assert_unsupported_error() {
  local file="$1"
  local label="$2"
  "$PYTHON_BIN" - "$file" "$label" <<'PY'
import json
import sys

file_path, label = sys.argv[1], sys.argv[2]
raw = open(file_path, "r", encoding="utf-8").read().strip()
data = json.loads(raw)
err = data.get("error")
if not err:
    raise SystemExit("missing error object")
msg = str(err.get("message", ""))
if err.get("code") not in (0, -32601):
    raise SystemExit(f"error code {err.get('code')} unexpected for {label}")
if "unsupported" not in msg and "Method not found" not in msg:
    raise SystemExit(f"error message {msg!r} unexpected for {label}")
PY
}

assert_tool_list_contains() {
  local file="$1"
  local tool="$2"
  "$PYTHON_BIN" - "$file" "$tool" <<'PY'
import json
import sys

file_path, want = sys.argv[1], sys.argv[2]
raw = open(file_path, "r", encoding="utf-8").read().strip()
data = json.loads(raw)
tools = data["result"]["tools"]
if not any(item.get("name") == want for item in tools):
    raise SystemExit(f"missing tool {want}")
PY
}

assert_tool_text_json_field() {
  local file="$1"
  local path="$2"
  local want="$3"
  local got
  got="$(tool_text_json_path "$file" "$path")"
  if [[ "$got" != "$want" ]]; then
    printf 'tool field %s = %s want %s\n' "$path" "$got" "$want" >&2
    exit 1
  fi
}

assert_tool_text_json_int_gt() {
  local file="$1"
  local path="$2"
  local min="$3"
  local got
  got="$(tool_text_json_path "$file" "$path")"
  if ! [[ "$got" =~ ^[0-9]+$ ]]; then
    printf 'tool field %s not an int: %s\n' "$path" "$got" >&2
    exit 1
  fi
  if (( got <= min )); then
    printf 'tool field %s = %s want > %s\n' "$path" "$got" "$min" >&2
    exit 1
  fi
}

run_endpoint() {
  local label="$1"
  local url="$2"
  local token="$3"

  printf '[%s] endpoint=%s\n' "$label" "$url"

  local init_body tools_body tools_resp status resp_file
  init_body='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"roots":{"listChanged":true}},"clientInfo":{"name":"tool-parity-smoke","version":"0.1.0"}}}'
  read -r status resp_file < <(call_rpc "$url" "$token" "$init_body")
  assert_status "$status" 200 "$label initialize"
  if [[ "$(json_path "$resp_file" 'result.protocolVersion')" != "2025-03-26" ]]; then
    printf '%s: protocolVersion mismatch\n' "$label" >&2
    exit 1
  fi
  if [[ "$(json_path "$resp_file" 'result.serverInfo.name')" != "hugo-mcp" ]]; then
    printf '%s: serverInfo.name mismatch\n' "$label" >&2
    exit 1
  fi
  if [[ "$(json_path "$resp_file" 'result.serverInfo.version')" != "1.0.0" ]]; then
    printf '%s: serverInfo.version mismatch\n' "$label" >&2
    exit 1
  fi

  tools_body='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  read -r status resp_file < <(call_rpc "$url" "$token" "$tools_body")
  assert_status "$status" 200 "$label tools/list"
  for tool in build_site create_page delete_page get_page list_assets list_pages update_page upload_asset; do
    assert_tool_list_contains "$resp_file" "$tool"
  done
  for tool in check_sri_versions generate_featured_image; do
    assert_tool_list_contains "$resp_file" "$tool"
  done

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_pages","arguments":{}}}')
  assert_status "$status" 200 "$label list_pages"
  assert_tool_text_json_int_gt "$resp_file" "total" -1

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_page","arguments":{"route":"_index","lang":"fr"}}}')
  assert_status "$status" 200 "$label get_page"
  assert_tool_text_json_field "$resp_file" "route" "/"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list_assets","arguments":{}}}')
  assert_status "$status" 200 "$label list_assets"
  assert_tool_text_json_int_gt "$resp_file" "count" -1

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"upload_asset","arguments":{"filename":"tool-parity-smoke.svg","data":"!!!"}}}')
  assert_status "$status" 200 "$label upload_asset"
  assert_tool_error "$resp_file" "Invalid base64"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"create_page","arguments":{"route":"/posts/tool-parity-smoke","title":"Smoke","content":"body","frontmatter":{"title":"Smoke"}}}}')
  assert_status "$status" 200 "$label create_page"
  assert_tool_error "$resp_file" "Conflict"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"update_page","arguments":{"route":"/posts/tool-parity-smoke-missing","lang":"fr","frontmatter":{"date":"2026-01-01T00:00:00Z"}}}}')
  assert_status "$status" 200 "$label update_page"
  assert_tool_error "$resp_file" "Page not found"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"delete_page","arguments":{"route":"/posts/missing","lang":"fr"}}}')
  assert_status "$status" 200 "$label delete_page"
  assert_tool_error "$resp_file" "Page not found"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"build_site","arguments":{"purge_cf":false}}}')
  assert_status "$status" 200 "$label build_site"
  if [[ "$(json_path "$resp_file" 'result.structuredContent.status')" != "built" ]]; then
    printf '%s: build_site did not return built\n' "$label" >&2
    exit 1
  fi

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"check_sri_versions","arguments":{"auto_fix":false,"dry_run":true}}}')
  assert_status "$status" 200 "$label check_sri_versions"
  sri_text="$(json_path "$resp_file" 'result.content.0.text')"
  if [[ "$sri_text" != *'"plugin":"sri-check"'* ]]; then
    printf '%s: check_sri_versions missing sri-check plugin marker\n' "$label" >&2
    exit 1
  fi
  if [[ "$sri_text" == *'"status":"no_handlers"'* ]]; then
    printf '%s: check_sri_versions still looks like the historical stub\n' "$label" >&2
    exit 1
  fi
  if [[ "$sri_text" != *'"dry_run":true'* ]]; then
    printf '%s: check_sri_versions did not echo dry_run=true\n' "$label" >&2
    exit 1
  fi

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"generate_featured_image","arguments":{"style":"tech","title":"Parity Tool","subtitle":"smoke","tags":["hugo","mcp"],"accent":"#7aa2f7","slug":"parity-tool-smoke","route":"/","lang":"fr"}}}')
  assert_status "$status" 200 "$label generate_featured_image"
  if [[ "$(json_path "$resp_file" 'result.content.0.text')" != *'"status":"ok"'* ]]; then
    printf '%s: generate_featured_image did not return ok\n' "$label" >&2
    exit 1
  fi

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":15,"method":"resources/list","params":{}}')
  assert_status "$status" 200 "$label resources/list"
  assert_result_list_or_error "$resp_file" "resources"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":16,"method":"prompts/list","params":{}}')
  assert_status "$status" 200 "$label prompts/list"
  assert_result_list_or_error "$resp_file" "prompts"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":17,"method":"does/not_exist","params":{}}')
  assert_status "$status" 200 "$label unknown method"
  assert_unsupported_error "$resp_file" "does/not_exist"

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}')
  assert_status "$status" 202 "$label notification without id"
  if [[ -s "$resp_file" ]]; then
    printf '%s: expected empty notification response\n' "$label" >&2
    exit 1
  fi

  read -r status resp_file < <(call_rpc "$url" "$token" '{"jsonrpc":"2.0","id":16,"method":"notifications/initialized","params":{}}')
  assert_status "$status" 200 "$label notification with id"
  assert_json_error "$resp_file" -32600 "unexpected id for"

  printf '[%s] ok\n' "$label"
}

main() {
  local runtime_url="${MCP_URL:-http://127.0.0.1:18182/mcp}"
  local runtime_token
  runtime_token="$(load_token "${MCP_TOKEN:-}" "${MCP_TOKEN_FILE:-}")" || {
    printf 'MCP_TOKEN or MCP_TOKEN_FILE is required\n' >&2
    exit 1
  }
  run_endpoint "runtime" "$runtime_url" "$runtime_token"

  if [[ -n "${SHIM_URL:-}" || -n "${SHIM_TOKEN:-}" || -n "${SHIM_TOKEN_FILE:-}" ]]; then
    local shim_url="${SHIM_URL:-http://192.168.122.69:18180/mcp}"
    local shim_token
    shim_token="$(load_token "${SHIM_TOKEN:-}" "${SHIM_TOKEN_FILE:-}")"
    run_endpoint "shim" "$shim_url" "$shim_token"
  fi

  printf 'tool-parity-smoke ok\n'
}

main "$@"
