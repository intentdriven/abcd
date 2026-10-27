---
schema_version: 1
id: "iss-2610251558011105"
slug: "docs-how-to-upgrade-to-v0-12-0-md-25-and"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D2"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/upgrade-to-v0.12.0.md"
remedy: "Label both as v0.12.0's: give the run log under ~/.abcd.noindex with a pointer to the v0.13.0 guide's rename, and say the next write is at the running abcd's current schema version (8 in v0.12.0)."
---

docs/how-to/upgrade-to-v0.12.0.md:25 and :38-41 state two v0.12.0 facts as if current. The run log is ~/.abcd.noindex/runs/<root-sha>/<date>.jsonl since v0.13.0 (internal/core/implement/run.go:41), and the next implement step or receipt writes the run state at the running binary's schema version, 13 in v0.15.1 and 14 on main, not 8 (internal/core/implement/loop/state.go:192-194). The page does not say either moved.
