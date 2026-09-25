---
schema_version: 1
id: "iss-134"
slug: "readdirbounded-determinism-doc-claim"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "iss-112/114/116 review (2026-07-24 run queue, burst 9)"
found_at: "internal/core/lifeboat/probe.go"
resolution: "readDirBounded's doc claims determinism only at or under the bound and says above it the surviving entries follow readdir order, with each caller's loud handling named; TestReadDirBoundedContract pins the narrowed contract. Deterministic selection above the bound was declined because it needs the whole listing."
impact: internal
resolved_by:
  commit: "ab9dd575"
---

readDirBounded's doc comment claims sorted-by-name determinism, which holds only at or under the bound; a directory exceeding the bound yields a readdir-order subset that is then sorted, so WHICH entries survive is nondeterministic (loud via truncated, consistent with ListDir, pathological trigger only)

## Grounds

- pursued: the doc states only what the function guarantees; shown wrong if readDirBounded returns other than bound sorted distinct entries with more=true above the bound, or the whole sorted directory at or under it, which the contract test asserts
