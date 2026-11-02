---
schema_version: 1
id: "iss-2610311604123786"
slug: "an-issue-lane-whose-widening-backed-off"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "building iss-2610311224444576's widening before an edit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/receipt.go"
remedy: "Record a widening's back-off on the lane (the path, the holding record and its expiry) and have the implement stage wait as a wait by design (exit 3, never counted as stuck) until that hold ends or is released, then hand the lane out again; this needs a run-state schema version"
---

An issue lane whose widening backed off (abcd implement widen, exit 3) has its implement stage handed out again at the very next step, with no wait for the hold that refused it to end: a host or runner stepping the run dispatches a fresh implementer that meets the same hold, as often as it steps, spending an agent each time on a lane that should wait for another copy's hold to end. A runner's end without a receipt is refused outside the step that counts refusals, so nothing bounds the repetition either.
