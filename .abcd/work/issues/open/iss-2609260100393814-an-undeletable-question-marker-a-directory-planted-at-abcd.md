---
schema_version: 1
id: "iss-2609260100393814"
slug: "an-undeletable-question-marker-a-directory-planted-at-abcd"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mode/question.go"
---

An undeletable question marker (a directory planted at .abcd/.work.local/question_open) makes every later prompt reset a hand-set mode to managed and repeat the error line: ResetOnAnswer writes managed first and only then fails to remove the marker, so the marker survives to trigger the same reset on the next message.
