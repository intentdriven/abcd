---
schema_version: 1
id: "iss-2610261739363732"
slug: "a-drain-lane-finished-outside-the"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-10-26 drain test lane"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/loop.go"
remedy: "Add a verb (implement adopt --run <id> --lane <n> --pr <number>, or a drain move that notices) which records an open or merged pull request on the lane's own branch as the lane's landing, checks that the branch and PR head match the lane and that the record is resolved there, sets the lane's outcome to pull-request, and lets the next drain move route it and open the next lane; and say in the drain's next line how to adopt or retire a lane rather than only how to drive it."
---

A drain lane finished outside the implement loop leaves the drain stuck: lane 1 of run-2610240852593097 (iss-2610142134353345) was refused at its implementer receipt on 2026-10-24, then finished by hand on 2026-10-26 (main merged in, record resolved, an Opus and a Sonnet review, a gated preflight, PR 1138 opened with Resolves). The run still says 'awaiting the implementer's receipt', and every abcd drain move answers 'drive the lane with implement step' and opens nothing more. No verb adopts a pull request opened by hand for a lane, or retires a lane whose work landed elsewhere; handing a receipt back would re-run the validators and open a second pull request, and a hand-back would discard the branch the open pull request is built on.
