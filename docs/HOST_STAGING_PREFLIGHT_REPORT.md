# Host Staging Preflight Report

## Scope

This report captures the preflight validation executed against the RC staging layout on `hugo-vm`.

## Preflight Command

The preflight script from the repo was executed on the VM staging layout as the `hugo-mcp` service identity:

```bash
sudo -u hugo-mcp env \
  PATH=/usr/local/bin:/usr/bin:/bin \
  HUGO_ROOT=/var/lib/hugo-mcp-go \
  HUGO_CONTENT_ROOT=/var/lib/hugo-mcp-go/content \
  HUGO_STATIC_ROOT=/var/lib/hugo-mcp-go/static \
  HUGO_EXPECTED_HUGO_BIN=/usr/local/bin/hugo \
  HUGO_EXPECTED_HUGO_VERSION_PREFIX="hugo v0.147.0-" \
  HUGO_SERVICE_USER=hugo-mcp \
  HUGO_SERVICE_GROUP=hugo-mcp \
  /tmp/preflight-hugo-mcp-go-run.sh
```

## Result

- exit status: `0`
- output: `preflight: ok`

## What Passed

- `HUGO_ROOT` exists and is canonical.
- `HUGO_CONTENT_ROOT` exists and is within `HUGO_ROOT`.
- `HUGO_STATIC_ROOT` exists and is within `HUGO_ROOT`.
- The three roots are not symlinks.
- The three roots are not group/other writable.
- The current user/group match `hugo-mcp:hugo-mcp`.
- `HUGO_EXPECTED_HUGO_BIN` exists, is executable, and is not a symlink.
- `hugo` resolves to `/usr/local/bin/hugo`.
- `hugo version` matches the expected prefix.

## Build Validation

Build and binary checks performed:

- local build:
  - `cd /home/jm/Documents/hugo-mcp-go && go build -o /tmp/hugo-mcp-go ./cmd/hugo-mcp-go`
- checksum:
  - local checksum: `89b642f5264b26a044621051a77934940b0300998f256cad72af49ca0189b2e2`
  - host checksum: matches the installed binary at `/srv/hugo-mcp-go/bin/hugo-mcp-go`
- runtime sanity:
  - the binary accepted the required staging env on the VM
  - it exited cleanly under a non-destructive timeout run

## Remaining Notes

- The preflight validates the staging layout only.
- It does not start the Go RC service.
- It does not modify the live Python service.
- It does not modify the NUC gateway.

