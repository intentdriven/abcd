---
schema_version: 1
id: "iss-2610130501353955"
slug: "the-cross-session-handover-index-was"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "session close-down after the v0.14.0 cut, 2026-10-13"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "When sessions close, abcd writes the handover index itself from the per-session handovers and the repository (every branch with owner, PR, commits ahead and merged state; a dependency and start order the sessions named), and names any session that closed without a handover, so the index never depends on one session surviving to write it"
---

The cross-session handover index was owed by one session, and when that session closed (on hitting the weekly limit) without writing its own handover or the index, nothing produced it: a person had to ask another session to reconstruct it on 2026-10-13, including three cloud runs only that session knew the state of (#974, #976, #982). The person asked that a future drain and similar runs do this automatically.
