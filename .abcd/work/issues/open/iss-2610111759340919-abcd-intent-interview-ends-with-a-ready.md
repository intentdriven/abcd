---
schema_version: 1
id: "iss-2610111759340919"
slug: "abcd-intent-interview-ends-with-a-ready"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "ruthless review, run-2610111657111142 lane-1, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_interview.go"
remedy: "Have the interview's end-of-interview verdict call intent.ReadyWithQuestions instead of intent.Ready, so it judges open questions as intent ready and the build do; test a planned record ending the interview with an open question, watched fail first."
---

abcd intent interview ends with a READY verdict computed by plain intent.Ready, so it can report 'the record is READY' for a planned record whose ## Open Questions still lists an unanswered item, which abcd intent ready now reports NOT READY (open_questions) and abcd build refuses at check (open_questions). Found by the ruthless review of run-2610111657111142 lane-1 (iss-2610101610446956).
