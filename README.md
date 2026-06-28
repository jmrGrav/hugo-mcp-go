# hugo-mcp-go

[![Go](https://img.shields.io/badge/go-1.25-blue)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/jmrGrav/hugo-mcp-go?label=release)](https://github.com/jmrGrav/hugo-mcp-go/releases/latest)
[![CI](https://github.com/jmrGrav/hugo-mcp-go/actions/workflows/ci.yml/badge.svg)](https://github.com/jmrGrav/hugo-mcp-go/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/jmrGrav/hugo-mcp-go)](LICENSE)

`hugo-mcp-go` is the Go implementation of the Hugo MCP backend used behind `mcp-runtime-go`.

It is operator-controlled, single-tenant, and designed to serve Hugo content tooling with a native HTTP backend while preserving the older stdio path for rollback.

## Architecture

```text
Claude / ChatGPT / Gemini / other MCP clients
  -> mcp-runtime-go (OAuth, gateway, public MCP surface)
  -> hugo-mcp-go (native HTTP backend)
  -> Hugo site content, assets, and build pipeline

Rollback paths preserved:
  -> hugo-mcp-shim
  -> legacy Python backend
```

`mcp-runtime-go` stays as the public OAuth and connector compatibility layer.
`hugo-mcp-go` provides the backend transport and tool execution.

## Tools

Read and chunked content tools:

- `list_pages`
- `get_page`
- `get_page_chunk`
- `list_assets`
- `get_asset_chunk`

Mutation and operational tools:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`
- `check_sri_versions`
- `generate_featured_image`

## Transport

- Native HTTP mode is explicitly enabled with `HUGO_MCP_TRANSPORT=http`
- `POST /mcp` is the compatibility endpoint
- stdio mode remains available for rollback and local use
- backend-only streaming events are available on `/mcp/events`
- tool catalogs now expose MCP `title` and tool annotations for classification

Example native backend URL:

- `http://127.0.0.1:18181/mcp`

## Configuration

Common required roots:

- `HUGO_ROOT`
- `HUGO_CONTENT_ROOT`
- `HUGO_STATIC_ROOT`

Native HTTP settings:

- `HUGO_MCP_TRANSPORT=http|stdio`
- `HUGO_MCP_HTTP_BIND_ADDR`
- `HUGO_MCP_HTTP_BIND_PORT`
- `HUGO_MCP_HTTP_TOKEN_FILE`
- `HUGO_MCP_STREAMING_ENABLED`
- `HUGO_MCP_MAX_CHUNK_BYTES`
- `HUGO_MCP_MAX_RESPONSE_BYTES`

## Security Model

- file-backed secrets only
- no shell execution
- bounded payload sizes
- traversal and symlink escape rejection
- explicit tool annotations for read-only versus destructive actions
- OAuth remains delegated to `mcp-runtime-go`

See:

- [`docs/ABOUT.md`](docs/ABOUT.md)
- [`SECURITY.md`](SECURITY.md)
- [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md)

## Schema Compatibility

All MCP tool `inputSchema` and `outputSchema` fragments are validated in the test suite to ensure they carry an explicit JSON Schema keyword (`type`, `oneOf`, `anyOf`, `allOf`, `$ref`, `enum`, or `const`). Empty fragments (`{}`) are rejected by the Claude Code MCP validator; the test `TestAllToolSchemasHaveNoEmptyFragments` catches any regression of this kind before it reaches production.

## Validation

The current release has been validated with:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `scripts/native-http-smoke.sh`

The native backend is live in controlled production, while the shim and Python backend remain preserved for rollback.

License: MIT
