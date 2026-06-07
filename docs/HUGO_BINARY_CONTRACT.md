# Hugo Binary Contract

## Purpose

Define the production contract for the `hugo` executable used by `hugo-mcp-go` during `build_site`.

## Recommended Path

- Recommended absolute path: `/usr/local/bin/hugo`

Rationale:

- explicit path beats ambient discovery
- locally pinned binaries are easier to audit than PATH-only discovery
- the path is stable across service restarts when managed by the deployment package

## Allowed PATH

If PATH lookup is retained for compatibility, it must be constrained to a short allowlist:

- `/usr/local/bin`
- `/usr/bin`
- `/bin`

`/usr/local/bin` must appear first if the release uses the recommended path.

## Startup Validation

Before starting the service, the deployment preflight must verify:

1. `hugo` resolves to the approved absolute path
2. `hugo version` is executable
3. the version string matches the approved release prefix
4. the binary is readable and executable by the service user

## If Hugo Is Missing

- fail closed
- do not start the service
- do not attempt an automatic fallback to another binary
- do not continue with build functionality disabled

## If Hugo Version Is Incompatible

- fail closed before service start
- require an explicit operator decision to approve the new version
- do not silently accept a newer or older `hugo`

## Operational Rule

The repository code still uses PATH lookup through `exec.CommandContext`. The production contract therefore depends on deployment-time validation and a pinned PATH or absolute binary path.

The release candidate does not change this runtime behavior; it makes the contract explicit for operations review.
