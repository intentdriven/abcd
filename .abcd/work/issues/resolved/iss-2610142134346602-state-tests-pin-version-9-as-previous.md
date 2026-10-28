---
schema_version: 1
id: "iss-2610142134346602"
slug: "state-tests-pin-version-9-as-previous"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "PR-976-review-round-5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state_text_test.go"
remedy: "Point the three tests at the version just before SchemaVersion (schemaVersionUnanswered, or a constant that always names SchemaVersion-1), keeping version 9 only where a test means version 9."
resolution: "The pace, receipt-text and hand-back-text state tests now read their earlier state at schemaVersionPrevious (SchemaVersion-1, today 14) instead of version 9, and a source check holds them to it across later bumps."
impact: internal
resolved_by:
  commit: "0e5575dc4a5510e4f28add290057f8ec0084058f"
---

Three tests (pace_stored_test.go, receipt_text_test.go, state_text_test.go) use schemaVersionUnkept, version 9, as the earlier version they migrate from, so since state schema version 11 they no longer test the version just before the current one.

## Grounds

- pursued: the three tests read a forged state at the current version and at SchemaVersion-1; it would be shown wrong if any of them still passed schemaVersionUnkept, or if schemaVersionPrevious stopped being the newest read-only version, either of which fails TestTheStoredTextTestsReadTheVersionJustBeforeTheCurrent
