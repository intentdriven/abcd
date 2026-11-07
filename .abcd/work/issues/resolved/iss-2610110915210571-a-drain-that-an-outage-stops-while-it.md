---
schema_version: 1
id: "iss-2610110915210571"
slug: "a-drain-that-an-outage-stops-while-it"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of PR #902 (itd-82 step 1, run-2610101724174688), round 2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "clear st.Judging when the outage stop ends the drain, or keep it and let a later --judgement be taken, with a test that stops a judging drain on an outage"
related_intents: [itd-2611041549461012]
resolution: "A drain that stops now clears the judgement it awaited in the move that ends it, so the state and the answer say the same; a later --judgement is told the drain awaits none. TestAnOutageStopReleasesTheJudgedIssue fails without the clear."
impact: fix
resolved_by:
  spec: "spc-2611041706004822"
  commit: "d7c05e9cb758ed0fe469a7a32c6d2c24cc0ffd06"
---

A drain that an outage stops while it waits on a host judgement keeps its judging entry: that move prints 'awaiting the host judgement' beside 'ended: the run gave up', and a --judgement given afterwards is refused. Nothing wrong is written, but the drain both claims to wait and refuses the answer.
