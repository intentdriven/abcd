---
schema_version: 1
id: "iss-2609252052381777"
slug: "intent-audit-owed-labels-every-entry-shipped-not-yet"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_drain.go"
---

intent audit --owed labels every entry 'shipped (not yet committed)' and omits shipped from the JSON when the history walk fails, so a day that is unknown reads exactly like an intent that is uncommitted
