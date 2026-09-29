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
resolution: "the widening-run summary checks the stand-down before counting an admission, so an admitted proposal with a contested, cyclic, unsafe or illegible disposition stands the run's summary down"
impact: fix
resolved_by:
  commit: "698f83df9"
---

The widening-run summary's stand-down is bypassed for an admitted item: internal/core/lint/readingoutstanding.go:421-425 takes the admissions.admits case before the unsafe/contested/cyclic check, so a run whose only acceptance is unreadable or contested still reports 'admitted 1, outstanding []', contradicting the type's own doc comment (lines 131-135; review-admission 1).

## Grounds

- pursued: a run whose admitted proposal carries an answer the walk cannot read reports no summary; a summary line reporting admitted 1 over a contested or illegible disposition would show it wrong
