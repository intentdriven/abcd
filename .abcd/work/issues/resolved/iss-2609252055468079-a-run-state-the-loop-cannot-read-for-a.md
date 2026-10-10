---
schema_version: 1
id: "iss-2609252055468079"
slug: "a-run-state-the-loop-cannot-read-for-a"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "readStateIn and runIDs refuse a state file or run directory the filesystem will not hand over (a symlink included) at the state step with the file and a remedy; TestASymlinkedRunStateIsRefusedInTheRefusalShape."
impact: fix
resolved_by:
  commit: "f27aa4bd"
---

A run state the loop cannot read for a filesystem reason, a symlinked state.json or a symlinked run-<id> directory under .abcd/.work.local/run/, fails closed but as a generic error (exit 1, 'reading …: not a regular file' or 'path escapes from parent') instead of the loop's refusal shape (step, reason, remedy; exit 2) that every other unreadable state takes (internal/core/implement/loop/state.go readStateIn and runIDs), so implement status/step/receipt and build report it with no remedy.

## Grounds

- pursued: we expect every unreadable run state to exit 2 in the refusal shape; shown wrong if a symlinked or unreadable state still surfaces as a generic exit 1
