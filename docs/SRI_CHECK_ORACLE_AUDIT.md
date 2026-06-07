# SRI Check Oracle Audit

Date: 2026-06-07

## Sources Audited

- Python oracle: `plugins/sri-check/plugin.py`
- Historical bash script on the VM: `/home/jm/scripts/check-sri-versions.sh`
- Hugo source tree used by the script:
  - `~/hugo-site/data/sri.yaml`
  - `~/hugo-site/assets/data/cdn/jsdelivr.yml`
  - `~/hugo-site/public`
  - `~/hugo-site`

## Script Responsibilities

The historical script is the operational oracle for the plugin. It does four things:

1. verifies tracked SRI hashes
2. checks whether jsDelivr-pinned libraries are behind the latest version
3. optionally auto-fixes same-major minor/patch updates
4. emits a final JSON report for MCP consumption

## Inputs and Flags

The script accepts these flags:

- `--json`
- `--no-autofix`
- `--no-cf-purge`
- `--dry-run`

Observed flag behavior:

- `--dry-run` implies `--no-autofix`
- `--no-autofix` suppresses file changes, rebuild, deploy, purge, and verification side effects
- `--no-cf-purge` keeps the auto-fix workflow but skips the Cloudflare purge step for orchestration by the caller
- `--json` appends a single JSON report after the textual log stream

## Files and Data Sources

### SRI verification source

- `~/hugo-site/data/sri.yaml`
- YAML entries are iterated as `url -> stored_sri`
- only string keys starting with `http` are considered

### CDN/version source

- `~/hugo-site/assets/data/cdn/jsdelivr.yml`
- the script reads `libFiles`
- each value is expected to contain a pinned jsDelivr path with a package name and a version

### Active package detection

- the script scans `~/hugo-site/public`
- it extracts `https://cdn.jsdelivr.net/npm/...` references from rendered site output
- only packages actually found in `public` are compared against the latest version API

## CDN and Library Patterns

The script looks for jsDelivr npm URLs in rendered output and CDN map entries in `jsdelivr.yml`.

Observed package parsing rules:

- npm package names may be scoped, for example `@scope/name`
- the pinned URL shape is `.../npm/<pkg>@<version>/...`
- inactive packages in the CDN map are skipped

## Version Logic

For each active package:

- fetch latest version metadata from `https://data.jsdelivr.com/v1/packages/npm/<pkg>`
- read `.tags.latest`
- compare the pinned version to the latest version
- if pinned equals latest, status is OK
- if latest sorts below or equal to the pinned version, status is OK
- if the latest version is newer and the major version is unchanged, the package is a minor/patch auto-fix candidate
- if the latest version is newer and the major version changes, the package becomes a manual-review warning

### Version source of truth

- the pinned version comes from `jsdelivr.yml`
- the latest version comes from the jsDelivr API

## SRI Logic

For each tracked SRI URL:

- fetch the live asset with `curl`
- compute `sha256` over the response body
- base64-encode the digest
- compare `sha256-<digest>` against the stored value

Behavior observed:

- a fetch failure is reported separately and is not treated as a successful SRI check
- a mismatch is always a warning
- SRI mismatches are never auto-fixed by the script

## Auto-Fix Behavior

Auto-fix only applies to same-major minor/patch version bumps.

Observed steps:

- back up `sri.yaml` and `jsdelivr.yml`
- update the pinned versions in `jsdelivr.yml`
- update matching SRI URLs in `sri.yaml`
- rebuild Hugo with `hugo --minify --cleanDestinationDir`
- deploy with `/home/jm/deploy.sh`
- optionally purge Cloudflare
- verify the live SRI hashes again
- roll back the files and deploy state if any step fails

### Files modified in auto-fix

- `~/hugo-site/assets/data/cdn/jsdelivr.yml`
- `~/hugo-site/data/sri.yaml`

### Cloudflare purge behavior

- default auto-fix purges Cloudflare with `purge_everything=true`
- `--no-cf-purge` skips the purge and expects the caller to orchestrate the purge
- the purge uses the Cloudflare v4 API and the VM-side token from the environment file

## Exit Codes

Observed exit contract:

- `0` when there are no remaining warnings after processing
- `1` when warnings remain after checks or after failed auto-fix rollback
- `2` when the required environment file cannot be read

## Notification Side Effects

Non-dry-run runs also manage operational notifications:

- on OK:
  - auto-resolve BetterStack incidents stored in `~/.config/sri-check.open-incidents`
  - ping the BetterStack heartbeat URL
- on WARN:
  - open a BetterStack incident
  - track the incident id in `~/.config/sri-check.open-incidents`

Dry-run suppresses those side effects.

## JSON Report Contract

When `--json` is present, the script prints a final marker and a single JSON object.

Top-level keys:

- `exit`
- `summary`
- `diagnostic`
- `auto_fix`
- `incident`
- `heartbeat_pinged`
- `dry_run`

Nested structure:

- `diagnostic.hash_mismatch`
- `diagnostic.major_outdated`
- `diagnostic.minor_outdated`
- `diagnostic.other`
- `auto_fix.ran`
- `auto_fix.applied`
- `auto_fix.failed`
- `auto_fix.skipped`
- `auto_fix.cf_purged`
- `incident.created`
- `incident.resolved`

## Oracle Verdict

- the Python plugin delegates to the historical script
- the native Go implementation must reproduce the JSON contract, version comparison behavior, SRI checks, and downstream hook trigger semantics
- the historical script is now an audit oracle only, not a runtime dependency for the Go backend
