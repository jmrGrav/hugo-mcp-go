# SHIM Host Install Report

## Scope

Install the VM-local `hugo-mcp-shim` staging artefacts without touching the live Python backend or the NUC gateway.

## Installed Artefacts

- `/srv/hugo-mcp-go/bin/hugo-mcp-shim`
- `/etc/systemd/system/hugo-mcp-shim.service`
- `/etc/hugo-mcp-go/hugo-mcp-shim.env`

## Permissions

- `/srv/hugo-mcp-go/bin/hugo-mcp-shim` `root:root 755`
- `/etc/systemd/system/hugo-mcp-shim.service` `root:root 644`
- `/etc/hugo-mcp-go/hugo-mcp-shim.env` `root:root 640`

## Local Build Artefact

- staging binary checksum: `aadab40715771e9b79130acba7aee0a76a942ea4b6510023b9bb521b73911dd6`

## Installation Notes

- the initial VM env file was corrected so `HUGO_MCP_GO_BIN` points to `/srv/hugo-mcp-go/bin/hugo-mcp-go`
- the systemd unit was corrected to allow the private bind address with `IPAddressAllow=192.168.122.69`
- the unit remains disabled; no enable action was taken

## Status After Install

- shim service: inactive
- service enablement: disabled
- Python live service: unchanged and still active on port `8000`
- shim port `18180`: closed after validation stop

## Rollback

- rollback action performed after validation: stop `hugo-mcp-shim.service` only
- no other services or gateway components were changed

## Verdict

- shim installed: yes
- ready to start in staging: yes
- ready to attach gateway now: no
