---
schema_version: 1
id: "iss-330"
slug: "parallel-mint-id-collision-observed-live"
severity: "minor"
category: "architectural-insight"
source: "user-observation"
found_during: "v0.6.1 re-cut"
resolution: "the capture mint is timestamp-numeric and collision-proof by construction: two minters at the same instant on different branches draw independent suffixes and cannot converge on one id; nothing renumbers"
impact: additive
resolved_by:
  intent: "itd-114"
  commit: "cf62b97"
---

Parallel-mint id collision observed live: the maintainer's bughunt session and the release session each minted iss-291..293 concurrently; the collision surfaced only at the merge-queue's record-lint on the second branch, costing a full release re-cut. Direct field evidence for itd-114 (collision-proof record ids) — the ledger's next-free-integer mint is unsafe under exactly the parallel-agent workflows the repo now runs daily