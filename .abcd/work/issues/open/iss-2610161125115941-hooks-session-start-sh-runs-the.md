---
schema_version: 1
id: "iss-2610161125115941"
slug: "hooks-session-start-sh-runs-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "v0.15.0 deferred findings, PR4, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/session-start.sh"
remedy: "When a plugin binary ran, exit with the entrypoints' status and report bootstrap's failure as a notice on stderr rather than as the exit code; test with a failing bootstrap.sh and a present binary that the hook exits 0 and the notice reaches stderr."
---

hooks/session-start.sh runs the session-start entrypoint even when hooks/bootstrap.sh failed but a plugin binary is present, then exits with bootstrap's non-zero code: the host shows its empty 'startup hook error' banner and drops the notices the entrypoint just printed, the outcome iss-2608241115201044 fixed for the entrypoint's own exit.
