---
schema_version: 1
id: "iss-2609251052592668"
slug: "refusesymlinkeddir-is-defined-twice-with-identical-bodies-in"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reading.go"
resolution: "One symlink guard: readingitem.RefuseSymlinkedDir, which capture's call sites reach under capture's own sentinel."
impact: internal
resolved_by:
  commit: "cba6b25afa923482425cadb5911d7e557494a833"
---

refuseSymlinkedDir is defined twice with identical bodies, in internal/core/capture/reading.go and internal/core/readingitem/readingitem.go, since the reading-item locator moved into the readingitem leaf; two copies of the symlink guard are how two walks come to disagree about what the ledger contains. One primitive, in the leaf, with capture's call sites routed through it.

## Grounds

- pursued: capture's refusal of a symlinked directory is the leaf's primitive and carries both sentinels with the message unchanged; TestSymlinkGuardIsTheLeafs would fail if a second copy returned
