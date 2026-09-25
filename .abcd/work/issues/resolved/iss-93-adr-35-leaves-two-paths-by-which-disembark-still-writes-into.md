---
schema_version: 1
id: "iss-93"
slug: "adr-35-leaves-two-paths-by-which-disembark-still-writes-into"
severity: "major"
category: "architectural-insight"
source: "agent-finding"
found_during: "itd-88-m0"
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md"
resolution: "decided: itd-88 and spc-3 shipped disembark read-only and out of tree (the voyage ledger, TestProbeLeavesEveryFileByteIdentical); no Pass-0 sync-stage or logbook writes remain"
impact: internal
resolved_by:
  intent: "itd-88"
---

adr-35 leaves two paths by which disembark still writes into the source repo: the Pass-0 dev-sync stage (writes .abcd/work/reviews, .abcd/memory, .abcd/work/issues into the repo) and the backgrounded-execution checkpoint (.abcd/logbook/disembark/<ts>/_state.json). adr-35 promises disembark is read-only over the source and that a test hashes the tree before and after, so both must move out-of-tree (under <dest>, or under ~/.abcd/voyage/<source-root-sha>/) or be excluded from the disembark path entirely. adr-35 does not settle where they go — a design decision owed before the packer ships.

## Grounds

- pursued: disembark writes nothing into the source repository; shown wrong if a disembark run changes any byte of the source tree
