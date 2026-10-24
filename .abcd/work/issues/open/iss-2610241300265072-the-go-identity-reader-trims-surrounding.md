---
schema_version: 1
id: "iss-2610241300265072"
slug: "the-go-identity-reader-trims-surrounding"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "review pair of feat/setup-redesign, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
remedy: "trim in the hook the same way, or refuse such a record in both readers with the same message"
---

the Go identity reader trims surrounding whitespace from the recorded name and email but the shell pre-commit gate's sed does not, so a hand-edited record with a leading or trailing space passes abcd ahoy --identity and blocks every commit in the hook (fails closed).
