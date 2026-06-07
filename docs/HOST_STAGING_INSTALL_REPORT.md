# Host Staging Install Report

## Scope

This report records the non-destructive staging-host installation of `hugo-mcp-go` on the real Hugo VM.

No cutover was performed.
The live Python service was not stopped.
The NUC gateway was not modified.

## Commands Executed

### Read-only preflight

- confirmed `hugo-mcp` user/group exist
- confirmed `/usr/local/bin/hugo`
- confirmed `hugo version`
- confirmed `/home/jm/hugo-site`
- confirmed no symlinks under the site tree
- confirmed `hugo-mcp.service` and `nginx.service` remain active

### Build and artifact staging

- `cd /home/jm/Documents/hugo-mcp-go && go build -o /tmp/hugo-mcp-go ./cmd/hugo-mcp-go`
- `scp /tmp/hugo-mcp-go hugo-vm:/tmp/hugo-mcp-go`
- `scp /home/jm/Documents/hugo-mcp-go/deploy/systemd/hugo-mcp-go.service hugo-vm:/tmp/hugo-mcp-go.service`
- `scp /home/jm/Documents/hugo-mcp-go/deploy/validation/preflight.sh hugo-vm:/tmp/preflight-hugo-mcp-go.sh`

### Staging layout creation

- created `/srv/hugo-mcp-go/bin`
- created `/etc/hugo-mcp-go`
- created `/var/lib/hugo-mcp-go`
- created `/var/lib/hugo-mcp-go/content`
- created `/var/lib/hugo-mcp-go/static`
- created `/var/lib/hugo-mcp-go/public`
- created `/var/lib/hugo-mcp-go/work`

### Non-destructive seed

- seeded `/var/lib/hugo-mcp-go` from `/home/jm/hugo-site`
- excluded `.git/` and `resources/_gen/`
- source tree was not modified
- rsync summary:
  - 5,525 files total
  - 4,761 regular files transferred
  - 67,832,688 bytes transferred

### Artifact installation

- installed `/srv/hugo-mcp-go/bin/hugo-mcp-go`
- installed `/etc/hugo-mcp-go/hugo-mcp-go.env`
- installed `/etc/systemd/system/hugo-mcp-go.service`
- did not enable the unit
- did not start the unit

## Final Permissions

- `/srv/hugo-mcp-go`: `root:root` `755`
- `/srv/hugo-mcp-go/bin`: `root:root` `755`
- `/srv/hugo-mcp-go/bin/hugo-mcp-go`: `root:root` `755`
- `/etc/hugo-mcp-go`: `root:root` `755`
- `/etc/hugo-mcp-go/hugo-mcp-go.env`: `root:root` `644`
- `/etc/systemd/system/hugo-mcp-go.service`: `root:root` `644`
- `/var/lib/hugo-mcp-go`: `hugo-mcp:hugo-mcp` `755`
- `/var/lib/hugo-mcp-go/content`: `hugo-mcp:hugo-mcp` `755`
- `/var/lib/hugo-mcp-go/static`: `hugo-mcp:hugo-mcp` `755`
- `/var/lib/hugo-mcp-go/public`: `hugo-mcp:hugo-mcp` `755`
- `/var/lib/hugo-mcp-go/work`: `hugo-mcp:hugo-mcp` `755`

## Installed Env

- `HUGO_ROOT=/var/lib/hugo-mcp-go`
- `HUGO_CONTENT_ROOT=/var/lib/hugo-mcp-go/content`
- `HUGO_STATIC_ROOT=/var/lib/hugo-mcp-go/static`
- `HUGO_MAX_REQUEST_BYTES=1048576`
- `HUGO_MAX_TOOL_ARGS_BYTES=262144`
- `HUGO_MAX_PAGE_BYTES=1048576`
- `HUGO_MAX_ASSET_BYTES=26214400`
- `HUGO_MAX_LIST_PAGES=500`
- `HUGO_MAX_LIST_ASSETS=500`
- `HUGO_EXPECTED_HUGO_BIN=/usr/local/bin/hugo`
- `HUGO_EXPECTED_HUGO_VERSION_PREFIX=hugo v0.147.0-`
- `HUGO_SERVICE_USER=hugo-mcp`
- `HUGO_SERVICE_GROUP=hugo-mcp`
- `PATH=/usr/local/bin:/usr/bin:/bin`

## Notes

- The unit file is installed but not enabled.
- The Go RC service is installed but not started.
- The live Python service on the VM remains untouched.
- The NUC gateway remains untouched.

