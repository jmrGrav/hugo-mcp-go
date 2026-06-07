# Second Canary Commands Dry Run

These commands are documentation only. Do not run them until the auth contract has been aligned.

## Assumptions

- Option A is selected.
- The shared token source is the NUC runtime env `HUGO_TOKEN`.
- The NUC source IP to the VM is `192.168.122.1`.

## 1. Back up the shim env

```bash
ssh hugo-vm 'sudo cp /etc/hugo-mcp-go/hugo-mcp-shim.env /etc/hugo-mcp-go/hugo-mcp-shim.env.bak.$(date +%F-%H%M%S)'
```

## 2. Align the shim backend token

```bash
RUNTIME_TOKEN=$(sudo awk -F= '/^HUGO_TOKEN=/{print $2}' /etc/mcp-runtime-go/mcp-runtime.env)
ssh hugo-vm "sudo perl -0pi -e 's|^HUGO_MCP_SHIM_BACKEND_TOKEN=.*$|HUGO_MCP_SHIM_BACKEND_TOKEN=$RUNTIME_TOKEN|m' /etc/hugo-mcp-go/hugo-mcp-shim.env"
```

## 3. Restart shim only

```bash
ssh hugo-vm 'sudo systemctl restart hugo-mcp-shim.service'
ssh hugo-vm 'sudo systemctl status --no-pager hugo-mcp-shim.service'
```

## 4. Direct NUC -> shim test with the runtime token

```bash
RUNTIME_TOKEN=$(sudo awk -F= '/^HUGO_TOKEN=/{print $2}' /etc/mcp-runtime-go/mcp-runtime.env)
curl -sk -D - \
  -H "Authorization: Bearer $RUNTIME_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}' \
  http://192.168.122.69:18180/mcp
```

```bash
curl -sk -D - \
  -H "Authorization: Bearer $RUNTIME_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","method":"tools/list","id":2,"params":{}}' \
  http://192.168.122.69:18180/mcp
```

## 5. Do not touch the gateway yet

- Do not change `HUGO_MCP_URL`.
- Do not restart `mcp-runtime.service`.
- Do not change OpenResty, Nginx, or Cloudflare.

## 6. Roll back the shim if needed

```bash
SHIM_BACKUP=$(ssh hugo-vm 'ls -1t /etc/hugo-mcp-go/hugo-mcp-shim.env.bak.* | head -1')
ssh hugo-vm "sudo cp \"$SHIM_BACKUP\" /etc/hugo-mcp-go/hugo-mcp-shim.env"
ssh hugo-vm 'sudo systemctl restart hugo-mcp-shim.service'
```
