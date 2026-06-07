# NUC to VM Network Audit

## Purpose

Document the real network path used for staging validation of `hugo-mcp-shim` on the VM.

## Hosts

- VM: `hugo-vm` at `192.168.122.69`
- NUC: `media-vm` at `192.168.122.187`

## Bridge / Subnet

- subnet: `192.168.122.0/24`
- gateway/router: `192.168.122.1`
- VM interface: `enp1s0`
- NUC interface: `enp1s0`

## VM Routing

- default route: `192.168.122.1 via enp1s0`
- local route: `192.168.122.0/24 dev enp1s0`

## Firewall State

### VM

- `firewalld`: inactive
- UFW: active
- default incoming policy: deny
- outgoing policy: allow
- existing allow rules:
  - `22/tcp`
  - `80/tcp`
  - `443/tcp`
  - `8000/tcp` from `192.168.122.1` for the live Python backend

### NUC

- no changes were made to the NUC firewall or gateway stack

## Shim Bind

- unit bind address: `192.168.122.69`
- shim port: `18180`
- Python backend port: `8000`

## Systemd Network Sandbox

- `IPAddressDeny=any`
- `IPAddressAllow=192.168.122.69`
- `IPAddressAllow=192.168.122.187`

## Temporary Validation Rules

- a temporary UFW allow rule was added on the VM for `192.168.122.187 -> 18180/tcp`
- that rule was used only for staging validation

## Observation

- ICMP from the NUC to the VM succeeds
- TCP to `18180` succeeds only after the temporary UFW allowance and the shim systemd allowlist include the NUC IP
- the live Python service on `8000` stays active throughout
