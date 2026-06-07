# Pre-Canary Review

## Incoherences Found

- Firewall source in the canary docs was inconsistent.
- The VM audit shows the NUC host as `192.168.122.187`, while `192.168.122.1` is the gateway/router.
- The canary firewall rule should therefore target source `192.168.122.187`, not `192.168.122.1`.
- The `initialize` payload was not captured in the local docs/tests, so reusing a made-up body would be unsafe.

## Corrections Applied

- Updated the canary docs to use source `192.168.122.187` for the temporary `18180/tcp` allow rule.
- Updated the rollback commands to remove the same `192.168.122.187` rule.
- Replaced the invented `initialize` body with an explicit instruction to reuse the previously validated payload or stop and recover it first.
- Added a short operator checklist in `docs/CANARY_READY_CHECKLIST.md`.

## Blockers Remaining

- The exact validated `initialize` payload is not embedded in the local docs.
- The manual canary should not start until that payload is available at execution time.
- No gateway canary has been run yet.

## Verdict

- Ready for manual canary: no
- Rollback ready: yes
- Cutover possible: non

