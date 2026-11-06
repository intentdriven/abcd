---
schema_version: 1
id: "iss-2611060835486997"
slug: "the-day-s-stop-page-is-rewritten-whole"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/stop.go"
remedy: "Note a checkout when a run starts there."
---

The day's stop page is rewritten whole from the checkouts it can list, this repository's worktrees and those a driver noted (`stop.go:928`), so a run in another checkout that never had a driver can drop off a later rewrite. A person relying on the page misses a run the stop did halt.
