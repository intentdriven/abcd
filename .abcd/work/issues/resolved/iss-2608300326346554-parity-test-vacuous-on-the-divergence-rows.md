---
schema_version: 1
id: "iss-2608300326346554"
slug: "parity-test-vacuous-on-the-divergence-rows"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-180 third-round ruthless review, 2026-08-30"
found_at: "internal/core/capture/standingparity_test.go, internal/core/capture/promote.go, internal/core/lint/readingoutstanding.go"
resolution: "The parity table now asserts lint's observable against the full standing set (proved by a newest-standing-wins mutation failing all three retires-nothing rows), the dispositions family root is Lstat-refused in capture's reader, and lint's read-only walk names the three ROOT-LEVEL trees it declined to enter instead of reporting them empty — an item's own disposition directory and a symlinked record file still read as unanswered until iss-2608300349493306."
impact: fix
---

The standing-disposition parity test cannot distinguish the converged reader from a divergent one on the three retires-nothing rows it was added for: when two records stand the expected id is empty, so both the outstanding and held consequences are false and a lint walk that stands only the newest record also reports neither — a mutation to newest-standing-wins passed all six rows. Assert lint's observable against the full standing set: an open hold exactly when the single standing record is a hold, none otherwise. Also, promote reads the standing disposition through a symlinked dispositions root (only the item leaf is Lstat-refused on that path), and lint's read-only walk follows or silently skips symlinks where capture refuses.
