---
schema_version: 1
id: "iss-2610142151424332"
slug: "take-back-asks-for-a-drop-not-needed"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "PR-976-final-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Have removePlaced also report whether an answer was recorded on the await, and when none was, skip the locked drop in takeBack so a busy lock refuses nothing and no message claims a recorded answer stays."
---

When no answer is recorded on the await, a concurrent invocation having already dropped it, and the run's lock is busy, takeBack still asks for the drop under the lock and its refusal says the recorded answer stays in the run's state, though there is none to drop.
