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
---

refuseSymlinkedDir is defined twice with identical bodies, in internal/core/capture/reading.go and internal/core/readingitem/readingitem.go, since the reading-item locator moved into the readingitem leaf; two copies of the symlink guard are how two walks come to disagree about what the ledger contains. One primitive, in the leaf, with capture's call sites routed through it.
