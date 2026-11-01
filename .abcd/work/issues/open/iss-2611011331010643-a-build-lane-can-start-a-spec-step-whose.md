---
schema_version: 1
id: "iss-2611011331010643"
slug: "a-build-lane-can-start-a-spec-step-whose"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "build run-2610312053537139 lane 2, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/spec/steps.go"
remedy: "Let a step's needs: line name another spec's step (spc-N step M) as well as its own, have the readiness check and build next refuse a step whose named step is not landed on the default branch, and have record-lint flag a 'Waits for spc-N' sentence under a step whose needs: line does not carry it."
---

A build lane can start a spec step whose wait on another spec's step is written only in prose, before that step has landed. A step's needs: line takes only the same spec's step numbers (internal/core/spec/steps.go), so a dependency across specs goes in prose. spc-2610301950037469 step 2 says it waits for spc-2610301917245442's steps 1 and 3; the readiness check cannot read that. On 2026-11-01 lane 2 of run-2610312053537139 started step 2 while those parent steps were still on an unmerged pull request (#1315). The implementer built the hook half, could not build the verb half, and the review returned FIX FIRST. Landing would have marked the whole step landed, and step 3, which reads the verb's limits field, would have started next. A full implement and preflight cycle was spent on a step that could not be finished.
