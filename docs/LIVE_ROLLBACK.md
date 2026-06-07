# Live Rollback

Objective: return the runtime to Python in under 2 minutes.

## Commands

```bash
sudo sed -i 's#^HUGO_MCP_URL=.*#HUGO_MCP_URL=https://192.168.122.69:8000/mcp#' /etc/mcp-runtime-go/mcp-runtime.env
sudo systemctl restart mcp-runtime.service
sudo systemctl --no-pager --full status mcp-runtime.service
```

Optional cleanup after the route flip:

```bash
sudo systemctl stop hugo-mcp-shim.service
sudo systemctl stop hugo-mcp-go.service
```

## Estimated Duration

- Route flip plus service restart: 30 to 90 seconds
- Optional cleanup of Go services: 10 to 20 seconds

## Post-rollback Checks

```bash
sudo systemctl is-active mcp-runtime.service
sudo grep '^HUGO_MCP_URL=' /etc/mcp-runtime-go/mcp-runtime.env
sudo journalctl -u mcp-runtime.service --since '5 minutes ago' --no-pager | tail -n 50
```

If a current OAuth access token is available, verify the MCP route with the same live smoke request set used before rollback.
