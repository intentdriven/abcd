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
resolution: "abcd intent interview's closing verdict now calls intent.ReadyWithQuestions, the verdict abcd intent ready reports, so a planned record left with an open question ends NOT READY with a failing open_questions row instead of READY."
impact: fix
resolved_by:
  commit: "44bd93f5c4078359bace0573e3d6e62f2b20e531"
---

abcd intent interview ends with a READY verdict computed by plain intent.Ready, so it can report 'the record is READY' for a planned record whose ## Open Questions still lists an unanswered item, which abcd intent ready now reports NOT READY (open_questions) and abcd build refuses at check (open_questions). Found by the ruthless review of run-2610111657111142 lane-1 (iss-2610101610446956).

## Grounds

- pursued: we expect the interview's verdict to agree with abcd intent ready and the build's check on every record it leaves; a record the interview calls READY that intent ready or the build refuses for open_questions would show it wrong.
