---
schema_version: 1
id: "iss-2610111759346608"
slug: "the-abcd-itd-n-dispatcher-computes-a"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "ruthless review, run-2610111657111142 lane-1, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/record/record.go"
remedy: "Compute the dispatcher's planned-intent next move from intent.ReadyWithQuestions, and when only open_questions fails point at the planning interview instead of implementing; test a planned intent with an open question, watched fail first."
duplicates: [iss-2610111759340919]
---

The abcd itd-N dispatcher computes a planned intent's next move from plain intent.Ready, so for a planned intent whose ## Open Questions still lists an unanswered item it says 'ready — implement against the spec body', while abcd intent ready reports NOT READY (open_questions) and abcd build refuses at check (open_questions). Found by the ruthless review of run-2610111657111142 lane-1 (iss-2610101610446956).
