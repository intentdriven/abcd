---
schema_version: 1
id: "iss-2609251842111593"
slug: "the-admission-ordering-gate-reads-committed-as-a-run-json"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/itemfate.go"
---

The admission ordering gate reads 'committed' as a run.json decoding to position comparative with a candidate_run, with no run_id/directory, manifest, item or git check (internal/core/capture/itemfate.go:145-186), so an untracked, id-less marker opens the gate; this matches the spec's Approach but not its line that no mutable file anywhere records the outcome (review-admission 2).
