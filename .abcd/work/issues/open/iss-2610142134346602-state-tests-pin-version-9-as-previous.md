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
---

Three tests (pace_stored_test.go, receipt_text_test.go, state_text_test.go) use schemaVersionUnkept, version 9, as the earlier version they migrate from, so since state schema version 11 they no longer test the version just before the current one.
