# oracle_phase2

This directory contains the mutation oracle capture for Phase 2 preparation.

Capture notes:

- tool schemas are captured from the Python source in `tool_schemas.json`
- nominal responses are recorded with `run_deploy()` normalized to `DEPLOY_SKIPPED`
- plugin hooks are stubbed to an empty list to avoid host-specific side effects
- side-effect snapshots store the resulting file contents after each nominal mutation

The capture is intentionally limited to the five mutation tools requested for Phase 2 preparation:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`
