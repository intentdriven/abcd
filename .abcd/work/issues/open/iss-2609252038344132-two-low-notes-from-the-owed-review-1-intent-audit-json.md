---
schema_version: 1
id: "iss-2609252038344132"
slug: "two-low-notes-from-the-owed-review-1-intent-audit-json"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/owed.go"
---

Two low notes from the owed review: (1) intent audit --json echoes a dead-letter reason's attacker-supplied text verbatim, so a quoted-back token shaped like 'Raw payload retained at .../.work.local/...' puts a local-tier-looking string in the output (the real retention path is cut correctly; internal/core/intent/owed.go:120-147), and the test's .work.local substring check proves the fixture, not the property; (2) the resolution of iss-2609100509537730 appends a near-duplicate pursued grounds bullet after its corroboration prose.
