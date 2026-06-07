# Toolchain Security Review

## Observed Toolchains

### Default shell toolchain

- `which go` -> `/usr/bin/go`
- `go version` -> `go version go1.25.0 linux/amd64`
- `go env GOVERSION` -> `go1.25.0`
- `go env GOTOOLCHAIN` -> `auto`
- `go env GOROOT` -> `/home/jm/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64`
- `PATH` includes `/usr/bin` before any manually installed tool bin directory

### Patched validation toolchain

- `GOTOOLCHAIN=go1.25.11 go version` -> `go version go1.25.11 linux/amd64`
- `GOTOOLCHAIN=go1.25.11 PATH="$HOME/go/bin:$HOME/.local/bin:$PATH" govulncheck ./...` -> no code vulnerabilities

## Security Finding

The default toolchain was too old for the current vulnerability database.
`govulncheck` reported 20 standard-library vulnerabilities when run against Go 1.25.0.
Those findings disappeared when the validation was forced onto Go 1.25.11.

## Conclusion

- Root cause: vulnerable Go toolchain version
- Corrective action: force or install Go 1.25.11 or newer for release validation
- Do not accept Go 1.25.0 for release validation

