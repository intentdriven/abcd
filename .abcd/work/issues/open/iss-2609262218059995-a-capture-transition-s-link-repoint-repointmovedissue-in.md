---
schema_version: 1
id: "iss-2609262218059995"
slug: "a-capture-transition-s-link-repoint-repointmovedissue-in"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainL"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
---

A capture transition's link repoint (repointMovedIssue in internal/core/capture/workflow.go) waits up to 5 s for the intent store's lock INSIDE the ledger lock, and the ledger's own acquisition budget (lockTimeout in internal/core/capture/alloc.go) is also 5 s, so an intent hold of 5 s or more makes a third process's ledger writer (a capture, a resolve, a link) fail with ErrAllocatorContention while the transition itself only reports a RelinkError. capture migrate --apply waits on the intent lock inside the ledger lock the same way. Bounded, not a deadlock: the two waits chain into a spurious failure on a slow machine.
