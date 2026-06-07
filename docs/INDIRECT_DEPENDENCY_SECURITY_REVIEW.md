# Indirect Dependency Security Review

## Scope

Review of the indirect dependency `golang.org/x/sys@v0.41.0` before the second canary.

## Origin

Commands executed:

```bash
go mod why -m golang.org/x/sys
go mod graph | grep 'golang.org/x/sys'
go list -m -versions golang.org/x/sys
```

Findings:

- `go mod why -m golang.org/x/sys`
  - `github.com/jmrGrav/hugo-mcp-go/internal/security/pathguard`
  - `golang.org/x/sys/unix`
- `go mod graph | grep 'golang.org/x/sys'`
  - root module depended on `golang.org/x/sys@v0.41.0`
  - `github.com/modelcontextprotocol/go-sdk@v1.6.1` also referenced `golang.org/x/sys@v0.41.0`
  - `golang.org/x/sys@v0.41.0` declared `go@1.24.0`
- `go list -m -versions golang.org/x/sys`
  - patch/fix version available
  - latest available at review time: `v0.45.0`

## Vulnerability

- ID: `GO-2026-5024`
- Package: `golang.org/x/sys/windows`
- Fixed in: `golang.org/x/sys@v0.44.0`
- Description: `NewNTUnicodeString` integer overflow in the Windows package

## Platforms

- Affected platform: Windows only
- Current release targets:
  - `linux/amd64`
  - `linux/arm64`
- Release builds do not produce Windows artifacts.

## Reachability

The Linux release path imports:

- `golang.org/x/sys/unix`
- `golang.org/x/sys/cpu`

It does not import `golang.org/x/sys/windows`.

`govulncheck -show verbose ./...` after the dependency bump reported no code vulnerabilities and no remaining reachable module vulnerability on the current Linux build path.

## Update Decision

Update applied: yes

Updated dependency:

- `golang.org/x/sys v0.41.0 -> v0.45.0`

Decision rationale:

- The vulnerability is Windows-only and not reachable in the current Linux release path.
- A patched version exists and the update is low risk.
- Keeping the dependency current removes the residual module finding and reduces future release friction if Windows support appears later.

## Validation Commands

Executed:

```bash
PATH="$HOME/go/bin:$HOME/.local/bin:$PATH" GOTOOLCHAIN=go1.25.11 go get golang.org/x/sys@latest
PATH="$HOME/go/bin:$HOME/.local/bin:$PATH" GOTOOLCHAIN=go1.25.11 go mod tidy
PATH="$HOME/go/bin:$HOME/.local/bin:$PATH" make release-check
PATH="$HOME/go/bin:$HOME/.local/bin:$PATH" GOTOOLCHAIN=go1.25.11 govulncheck -show verbose ./...
```

Result:

- `make release-check`: green
- `govulncheck`: no reachable vulnerabilities in code
- `gitleaks`: no leaks found
- `trufflehog`: `verified_secrets=0`, `unverified_secrets=0`
- local and multi-arch builds: passed
- `sha256sum -c dist/SHA256SUMS`: passed

## Final Decision

- Update applied: yes
- Risk accepted: no, because the dependency was remediated instead of waived
- Release pipeline green: yes
- Blocker before second canary: no

