---
schema_version: 1
id: "iss-2610252214325849"
slug: "found-in-a-counterpart-internal-surface"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard.go"
remedy: "Change the comment to exit 1 and give the stderr-discard reason."
---

Found in a counterpart: internal/surface/cli/guard.go:31-34 says the hook treats a guard that cannot answer as 'fail-OPEN (loud, exit 0 ...)'. failOpen returns exit 1 (guard.go:332-335), as 17-guard.md:71-73 and docs/how-to/install.md:152-157 state, because the host discards stderr on exit 0.
