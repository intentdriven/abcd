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
---

A drain that an outage stops while it waits on a host judgement keeps its judging entry: that move prints 'awaiting the host judgement' beside 'ended: the run gave up', and a --judgement given afterwards is refused. Nothing wrong is written, but the drain both claims to wait and refuses the answer.
