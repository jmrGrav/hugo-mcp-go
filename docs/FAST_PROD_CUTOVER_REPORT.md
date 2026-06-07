# FAST Prod Cutover Report

Date: 2026-06-07

## Summary

`hugo-mcp-go` is deployed on the VM backend path used by `mcp-runtime.service`.
The active backend path is now native HTTP on `192.168.122.69:18181`; the shim and Python backends remain installed for rollback.

## Commit Deployed

- `5ff3763` (`mcp: add native http transport and tool metadata`)

## Backend URLs

- Previous Python backend URL: `http://192.168.122.69:8000/mcp`
- Previous shim rollback backend URL: `http://192.168.122.69:18180/mcp`
- Native HTTP backend URL used by the runtime path: `http://192.168.122.69:18181/mcp`

## Service State

- `mcp-runtime.service`: active on the host, listening on `127.0.0.1:8086`
- `hugo-mcp-go-http.service`: active on `hugo-vm`, listening on `192.168.122.69:18181`
- `hugo-mcp-shim.service`: preserved on `hugo-vm` for rollback
- `hugo-mcp-go.service`: preserved on `hugo-vm` for stdio/rollback
- Python backend `hugo-mcp.service`: still active on `hugo-vm` for rollback

## Environment

- `HUGO_MCP_URL` is set to `http://192.168.122.69:18181/mcp`
- rollback remains the Python URL in `/etc/mcp-runtime-go/mcp-runtime.env` backups

## Commands Used

```bash
go build -o /tmp/hugo-mcp-go ./cmd/hugo-mcp-go
go build -o /tmp/hugo-mcp-shim ./cmd/hugo-mcp-shim
scp /tmp/hugo-mcp-go hugo-vm:/tmp/hugo-mcp-go
scp /tmp/hugo-mcp-shim hugo-vm:/tmp/hugo-mcp-shim
ssh hugo-vm 'sudo install -o root -g root -m 0755 /tmp/hugo-mcp-go /srv/hugo-mcp-go/bin/hugo-mcp-go && sudo install -o root -g root -m 0755 /tmp/hugo-mcp-shim /srv/hugo-mcp-go/bin/hugo-mcp-shim && sudo systemctl restart hugo-mcp-shim.service'
sudo systemctl restart mcp-runtime.service
```

## Validation Performed

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `scripts/tool-parity-smoke.sh`
- `scripts/hooks-smoke.sh`
- direct VM backend calls against `http://192.168.122.69:18181/mcp`

## Direct VM Smoke Results

Bearer used:

- the shared runtime token from `/etc/mcp-runtime-go/mcp-runtime.env`

Results:

- `initialize`: success
- `tools/list`: success
- `list_pages`: success
- `get_page`: success
- `check_sri_versions` in safe dry-run mode: success
- `build_site`: success, with structured response showing `status: built`

Observed tool catalog on the VM backend now includes:

- `check_sri_versions`
- `generate_featured_image`

After the metadata redeploy, the live native catalog also exposes:

- `title`
- `annotations.readOnlyHint`
- `annotations.destructiveHint`

Examples from `tools/list` on the native backend after redeploy:

- `list_pages` -> `title: List pages`, `annotations.readOnlyHint: true`
- `create_page` -> `title: Create page`, `annotations.destructiveHint: true`

## Gateway Smoke Status

- `mcp-runtime.service` was restarted and is active
- direct gateway calls to `http://127.0.0.1:8086/mcp` with `HUGO_TOKEN` still return `401 Unauthorized`, which is expected because `HUGO_TOKEN` is a backend token, not an OAuth client bearer
- this shell session did not obtain an OAuth access token from the authorized client flow, so an end-to-end gateway smoke through the authenticated client was not completed here

## Logs

- VM shim logs remained redacted
- no bearer token or raw request body appeared in the captured journald output
- no unexpected 5xx burst was observed during the direct VM backend checks
- the local unauthenticated gateway probe returned `401` with `Bearer token required`; there was no log evidence of an IP-denied condition in this session

## Rollback

- rollback ready: yes
- rollback target: restore `/etc/mcp-runtime-go/mcp-runtime.env` to the Python URL backup and restart `mcp-runtime.service`
- Python preserved: yes

## Verdict

- Go branchée en production: yes
- `mcp-runtime-go` pointe vers native HTTP Go: yes
- Python conservé: yes
- rollback prêt: yes
- errors restantes: gateway OAuth bearer not captured in this shell session

## Next Minimum Action

- obtain a real OAuth bearer from the authorized client path and re-run `initialize`, `tools/list`, `list_pages`, `get_page`, and `check_sri_versions` through `mcp-runtime.service`

## Claude.ai Tool Refresh Incident

- timestamp: 2026-06-07 14:55:25-14:55:26 CEST, based on VM journald entries observed immediately after the click
- visible UI state: Claude showed `Impossible de recharger les outils depuis le serveur` while still listing 10 Hugo tools in the connector panel
- host gateway logs: no matching `mcp-runtime.service` request lines were emitted in the observed window
- VM shim logs: `initialize` returned `200`, `notifications/initialized` returned `202`, and `tools/list` returned `200` with the Go tool catalog
- proxy logs: no matching `/mcp`, `/authorize`, `/token`, `/.well-known`, or `/register` entries were present in the local nginx logs checked here
- classification: UI/client-side refresh error after a successful backend refresh sequence; no server-side failure was proven in this session
- root cause: not proven server-side; the VM backend refreshed successfully, so the visible error is more consistent with a Claude-side refresh/state problem than with a Go backend failure
- action taken: captured host, VM, and proxy logs; no code changes; no rollback
- next action: retry the refresh from the authorized Claude client and, if it fails again, capture the exact click time plus any browser/network console evidence
- catalog note: live Go/shim catalog exposes 12 tools, live Python exposes 10 tools, and the missing Python-era tools are `get_page_chunk` and `get_asset_chunk`
- permissions note: the live Go/shim catalog captures in this session did not expose MCP `title` or `annotations`; the repo now adds those hints, but they still need deployment before Claude can benefit from them
- operator validation: Claude production validation reached the 11/11 controlled-tool set, including `build_site`, `generate_featured_image`, `get_page_chunk`, and `get_asset_chunk`, with no shim or Python traffic observed

## Native HTTP Transport Cutover

- native HTTP transport implemented: yes
- backend SSE capability: yes, optional and backend-only
- chunking/pagination for large content: yes
- stdio preserved: yes
- native HTTP smoke script: `scripts/native-http-smoke.sh`
- native HTTP service packaging: `deploy/systemd/hugo-mcp-go-http.service`
- shim still present for rollback: yes
- Python still present for rollback: yes
- `mcp-runtime-go` modified in this mission: no
- host-to-VM port 18181 access: yes, via UFW allow from `192.168.122.1`
- service supplementary group fix: yes, `hugo-mcp-shim` added so the service can read existing content/static files
- public build output ownership fix: yes, `/var/lib/hugo-mcp-go/public` now owned by `hugo-mcp:hugo-mcp`
- cutover time: 2026-06-07 18:05 CEST
- old rollback backend URL: `http://192.168.122.69:18180/mcp`
- active native backend URL: `http://192.168.122.69:18181/mcp`
- cutover validated on VM/native HTTP path: yes
- native smoke result: passed
- rollback ready: yes
- public POST /mcp probe: `401 Bearer token required`
- public GET /mcp probe: `405 Not Allowed`
- Claude.ai refresh result: pending live operator/client refresh after gateway cutover
- logs confirming native backend calls: `hugo-mcp-go-http.service` shows successful `POST /mcp` requests during native smoke and direct probes
- logs confirming absence of shim calls: no shim calls were observed during the native smoke window
- verdict: native HTTP transport is active on the runtime path; the shim and Python remain rollback-only
