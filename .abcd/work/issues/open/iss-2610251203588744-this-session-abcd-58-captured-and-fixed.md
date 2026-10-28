---
schema_version: 1
id: "iss-2610251203588744"
slug: "this-session-abcd-58-captured-and-fixed"
severity: "minor"
category: "lapse"
source: "agent-observation"
found_during: "preflight of fix/unit-stage-budget, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
lapsed_at: "2026-10-25T07:24:15Z"
remedy: "Before capturing a failure seen in a shared gate, run go run ./cmd/abcd peers and search the peer worktrees' ledgers for the failing test's name. AGENTS.md's concurrent-sessions rule already requires the scan before capturing; this record notes that it lapsed."
---

This session (abcd-58) captured and fixed three load-sensitive test failures as three new records on fix/unit-stage-budget, without first running abcd peers or checking the peer branches. A peer's chore/capture-holdups-3fee already held the same finding, iss-2610241911001655, reserved at 10:01 and fixed in the same three files. So the work was done twice, and the two branches would have conflicted. The duplicates were dropped before anything was pushed.
