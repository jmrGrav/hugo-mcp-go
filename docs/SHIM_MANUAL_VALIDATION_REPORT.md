# SHIM Manual Validation Report

## Commands Executed

- local build and test:
  - `go test ./...`
  - `go test -race ./...`
  - `go vet ./...`
  - `go build -o /tmp/hugo-mcp-shim ./cmd/hugo-mcp-shim`
  - `go test ./internal/... -coverprofile=coverage.out`
  - `go tool cover -func=coverage.out | tail -1`
- VM install and validation:
  - copied `/tmp/hugo-mcp-shim` to the VM and installed it as `/srv/hugo-mcp-go/bin/hugo-mcp-shim`
  - `systemctl daemon-reload`
  - `systemctl start hugo-mcp-shim.service`
  - `systemctl stop hugo-mcp-shim.service`
- HTTP validation on the VM:
  - `POST /mcp` with no token
  - `POST /mcp` with invalid token
  - `POST /mcp` with malformed JSON
  - `initialize`
  - `notifications/initialized`
  - `tools/list`
- direct NUC validation attempt:
  - attempted SSH access to `192.168.122.1`

## Results

- no token -> rejected with `401`
- invalid token -> rejected with `401`
- malformed JSON -> rejected with `400`
- `initialize` -> accepted with `200`
- `notifications/initialized` -> accepted with `202`
- `tools/list` after initialization -> returned the Go tool catalog with `200`

## Files Installed

- `/srv/hugo-mcp-go/bin/hugo-mcp-shim`
- `/etc/systemd/system/hugo-mcp-shim.service`
- `/etc/hugo-mcp-go/hugo-mcp-shim.env`

## Final Host State

- shim service: inactive
- shim service enablement: disabled
- Python live service: still active
- NUC and gateway: unchanged

## Direct NUC Attempt

- SSH to `192.168.122.1` was reachable at the TCP layer but public-key authentication was denied from this session
- no direct `curl` could be executed from the NUC side in this environment

## Anomalies

- the staging env file initially pointed `HUGO_MCP_GO_BIN` at the shim binary instead of the stdio backend; that was corrected
- the systemd network sandbox initially blocked TCP reply traffic until `IPAddressAllow=192.168.122.69` was added

## Verdict

- Python intact: yes
- NUC intact: yes
- shim installed: yes
- shim started staging: yes, then stopped
- shim responds locally: yes
- shim reachable from the NUC directly: not confirmed from this session
- gateway attachable now: no
- cutover: no
