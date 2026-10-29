---
schema_version: 1
id: "iss-2610270934594826"
slug: "abcd-drain-unskip-names-the-wrong-thing"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "itd-82 lane-1 round-2 review, 2026-10-27"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "When an --unskip id names no entry but a held-back path covers it, say that the issue is held back by that path and name the --unskip that lifts it, rather than that it was not held back."
---

abcd drain --unskip names the wrong thing when the issue it is given is held back by a path: with internal/core held back, --unskip iss-A for an issue whose found_at lies in internal/core says iss-A was not held back, while the same move lists iss-A among the held-back issues (applySkips in internal/core/implement/loop/drain_skip.go). The command that would lift the hold, --unskip internal/core, is still shown, so nothing is lost, but the summary line is false. Found by the round-2 ruthless review of itd-82 lane-1 (build/run-2610260500113538-lane-1).
