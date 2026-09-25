---
schema_version: 1
id: "iss-2609251707308675"
slug: "changelog-sentence-never-refuses"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
---

The abcd changelog sentence says it never refuses, but the verb exits 2 outside a checkout or on an unreadable ledger (emitCut error in ship.go)
