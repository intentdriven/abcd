---
schema_version: 1
id: "iss-2609251842112266"
slug: "the-widening-run-summary-s-stand-down-is-bypassed-for-an"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/readingoutstanding.go"
---

The widening-run summary's stand-down is bypassed for an admitted item: internal/core/lint/readingoutstanding.go:421-425 takes the admissions.admits case before the unsafe/contested/cyclic check, so a run whose only acceptance is unreadable or contested still reports 'admitted 1, outstanding []', contradicting the type's own doc comment (lines 131-135; review-admission 1).
