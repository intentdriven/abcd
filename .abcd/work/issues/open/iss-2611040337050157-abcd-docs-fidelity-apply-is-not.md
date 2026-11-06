---
schema_version: 1
id: "iss-2611040337050157"
slug: "abcd-docs-fidelity-apply-is-not"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/docfidelity"
remedy: "Skip a sentence whose replacement already stands on that line and a flag already recorded for that commit and sentence; a test applies twice and asserts one result."
---

abcd docs fidelity --apply is not idempotent: run twice on 2026-11-04, a replacement that begins with the sentence it replaces is matched and applied again, doubling the clause in the brief and duplicating its flag.
