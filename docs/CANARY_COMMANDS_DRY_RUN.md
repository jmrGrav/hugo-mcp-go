# Canary Commands Dry Run

## Scope

These are the commands for a future canary. The `initialize` body must be taken from the previously validated payload and is not re-derived here. Do not run them as part of this documentation task.

## 0. Record the Current State

### NUC

```bash
systemctl status --no-pager mcp-runtime.service
sudo -n cat /etc/mcp-runtime-go/mcp-runtime.env
```

### VM

```bash
ssh hugo-vm 'systemctl status --no-pager hugo-mcp.service hugo-mcp-shim.service'
ssh hugo-vm 'sudo -n cat /home/jm/hugo-mcp/.env'
ssh hugo-vm 'sudo -n cat /etc/hugo-mcp-go/hugo-mcp-shim.env'
ssh hugo-vm 'sudo -n nft list ruleset'
```

## 1. Backup the Current Gateway Env

### NUC backup

```bash
ts=$(date +%F-%H%M%S)
sudo cp -a /etc/mcp-runtime-go/mcp-runtime.env \
  /etc/mcp-runtime-go/mcp-runtime.env.bak.$ts
```

Practical note:

- capture the timestamp once and reuse it for restore
- do not hand-edit the backup file

### VM backup

```bash
ssh hugo-vm 'ts=$(date +%F-%H%M%S); sudo -n cp -a /home/jm/hugo-mcp/.env /home/jm/hugo-mcp/.env.bak.$ts; sudo -n cp -a /etc/hugo-mcp-go/hugo-mcp-shim.env /etc/hugo-mcp-go/hugo-mcp-shim.env.bak.$ts'
```

## 2. Start the Shim

### VM

```bash
ssh hugo-vm 'sudo -n systemctl start hugo-mcp-shim.service'
ssh hugo-vm 'systemctl status --no-pager hugo-mcp-shim.service'
ssh hugo-vm 'journalctl -u hugo-mcp-shim.service -n 80 --no-pager'
```

## 3. Temporary Firewall Allow Rule

### VM

```bash
ssh hugo-vm 'sudo -n ufw allow from 192.168.122.187 to any port 18180 proto tcp comment "hugo-mcp canary to shim"'
ssh hugo-vm 'sudo -n ufw status numbered'
```

Rollback of the temporary rule:

```bash
ssh hugo-vm 'sudo -n ufw delete allow from 192.168.122.187 to any port 18180 proto tcp'
```

## 4. Direct NUC -> Shim Test

### Export the shim token from the protected env file

```bash
export HUGO_MCP_SHIM_BACKEND_TOKEN="$(ssh hugo-vm 'sudo -n awk -F= '\''/^HUGO_MCP_SHIM_BACKEND_TOKEN=/{print $2}'\'' /etc/hugo-mcp-go/hugo-mcp-shim.env')"
```

### Initialize

Use the payload validated during the previous session. Do not invent a new `initialize` body here because the validated JSON was not captured in the local docs/tests.

If that payload is not available at execution time, stop and recover it from the prior validation session before continuing.

### Tool listing

```bash
curl -sS -D - \
  -H "Authorization: Bearer ${HUGO_MCP_SHIM_BACKEND_TOKEN}" \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  http://192.168.122.69:18180/mcp
```

## 5. Temporary Gateway Switch

### Backup and edit the NUC env file

```bash
sudo cp -a /etc/mcp-runtime-go/mcp-runtime.env \
  /etc/mcp-runtime-go/mcp-runtime.env.canary
sudo perl -0pi -e 's|^HUGO_MCP_URL=.*$|HUGO_MCP_URL=http://192.168.122.69:18180/mcp|m' \
  /etc/mcp-runtime-go/mcp-runtime.env
grep '^HUGO_MCP_URL=' /etc/mcp-runtime-go/mcp-runtime.env
```

### Reload and restart the gateway

```bash
sudo systemctl daemon-reload
sudo systemctl restart mcp-runtime.service
sudo systemctl status --no-pager mcp-runtime.service
sudo journalctl -u mcp-runtime.service -n 100 --no-pager
```

### Confirm the live route

```bash
sudo ss -ltnp | grep ':8086'
```

## 6. Verification After the Switch

Use the existing authenticated MCP client for the real gateway path, then watch journald on both hosts.

### NUC

```bash
sudo journalctl -fu mcp-runtime.service
```

### VM

```bash
ssh hugo-vm 'sudo -n journalctl -fu hugo-mcp-shim.service'
ssh hugo-vm 'sudo -n journalctl -fu hugo-mcp.service'
```

## 7. Rollback to Python

### Restore the gateway env

```bash
sudo cp -a /etc/mcp-runtime-go/mcp-runtime.env.canary \
  /etc/mcp-runtime-go/mcp-runtime.env
sudo systemctl daemon-reload
sudo systemctl restart mcp-runtime.service
sudo grep '^HUGO_MCP_URL=' /etc/mcp-runtime-go/mcp-runtime.env
```

Fallback:

- if `/etc/mcp-runtime-go/mcp-runtime.env.canary` is missing, restore from the timestamped backup created in step 1 instead

### Clean up the canary rule

```bash
ssh hugo-vm 'sudo -n ufw delete allow from 192.168.122.187 to any port 18180 proto tcp'
```

### Stop the shim

```bash
ssh hugo-vm 'sudo -n systemctl stop hugo-mcp-shim.service'
ssh hugo-vm 'systemctl status --no-pager hugo-mcp-shim.service'
```

## 8. Final Sanity Checks

```bash
systemctl status --no-pager mcp-runtime.service
ssh hugo-vm 'systemctl status --no-pager hugo-mcp.service hugo-mcp-shim.service'
ssh hugo-vm 'sudo -n ss -ltnp | grep -E "(:8000|:18180)\\b"'
```
