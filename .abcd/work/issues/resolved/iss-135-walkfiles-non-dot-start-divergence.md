---
schema_version: 1
id: "iss-135"
slug: "walkfiles-non-dot-start-divergence"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "iss-112/114/116 review (2026-07-24 run queue, burst 9)"
found_at: "internal/core/lifeboat/probe.go"
resolution: "walkStart classifies a non-dot start component by component (lstat only): skip-set or symlinked components yield nothing, a regular-file leaf yields itself, a directory leaf is descended; TestWalkFilesStartBoundaryMatchesTheWholeWalk checks each start against the whole-tree walk."
impact: internal
resolved_by:
  commit: "f85da95a"
---

WalkFiles start-boundary divergence for a non-dot start: a start dir named in the skip set would now be walked and a regular-file start returns nil; unreachable today (both callers pass dot) — flag for any future caller passing a non-dot start

## Grounds

- pursued: any start below the root yields exactly the whole walk's subset beneath it; shown wrong if a start path yields a file the whole-tree walk does not, which the test asserts for every case
