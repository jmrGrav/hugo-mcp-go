# Host Cutover Readiness

## Decision Summary

The real infrastructure is **not yet ready** for a cutover of `hugo-mcp-go`.

## Prerequisites Checked

### Validated

- The Hugo VM has a usable `hugo` binary at `/usr/local/bin/hugo`.
- The Hugo VM has the real site tree at `/home/jm/hugo-site`.
- The Hugo VM has the dedicated `hugo-mcp` user and group available.
- The NUC has the gateway service `mcp-runtime.service`.
- The NUC has OpenResty active.
- The Hugo VM has nginx active.
- The site tree contains `content/`, `static/`, `public/`, and `themes/`.
- No symlinks were found in the Hugo site tree.
- The RC staging layout now exists at `/var/lib/hugo-mcp-go`.
- The RC env file now exists at `/etc/hugo-mcp-go/hugo-mcp-go.env`.
- The RC unit file now exists at `/etc/systemd/system/hugo-mcp-go.service`.
- The RC preflight passes on the staging layout.
- The staged binary exists at `/srv/hugo-mcp-go/bin/hugo-mcp-go`.
- The VM-local shim staging package is installed at `/srv/hugo-mcp-go/bin/hugo-mcp-shim` and validates locally on `192.168.122.69:18180`.
- The NUC-to-shim staging path is validated end to end from `192.168.122.187`.

### Missing

- The Go RC service has not been started yet.
- The current live backend on the VM is still Python/HTTP.
- The gateway/backend attachment for the Go RC is not yet installed.
- The current live service on the VM still runs as `jm`.
- A runtime validation of the Go RC as a live service has not been performed yet.
- The NUC gateway remains unchanged.

## Risks

- Transport mismatch between the current live Python backend and the Go RC contract.
- Host split between the NUC gateway and the VM site service.
- The Go RC is staged but not live, so the transport has not been exercised end to end.
- The current live service identity is not the dedicated service account from the RC package.

## Readiness Verdict

- Ready to install on the host: **yes**
- Ready to start in staging on the host: **yes**
- Ready to attach to the gateway: **no**
- Ready to cut over: **no**

## Minimal Remaining Actions

1. Install the RC package layout on the VM.
2. Start the Go RC in staging only after operator approval.
3. Validate the gateway/backend attachment model separately.
4. Re-run the staging smoke checks before any cutover discussion.
