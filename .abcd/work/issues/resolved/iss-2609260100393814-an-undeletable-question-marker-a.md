---
schema_version: 1
id: "iss-2609260100393814"
slug: "an-undeletable-question-marker-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mode/question.go"
resolution: "ResetOnAnswer refuses a directory at the marker's path before writing anything, so a hand-set mode survives every message and each message prints one line naming the path; its errors say whether the mode moved."
impact: internal
resolved_by:
  commit: "4490a2ac"
---

An undeletable question marker (a directory planted at .abcd/.work.local/question_open) makes every later prompt reset a hand-set mode to managed and repeat the error line: ResetOnAnswer writes managed first and only then fails to remove the marker, so the marker survives to trigger the same reset on the next message.

## Grounds

- pursued: with a planted directory at the marker, two consecutive prompts leave a hand-set facilitator mode as it was with one abcd mode line each; shown wrong if TestPromptHookLeavesAHandSetModeOverAnUnremovableMarker sees the mode move to managed
