---
schema_version: 1
id: "iss-2610230856272073"
slug: "a-dependent-build-lane-waits-for-its"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "lane C, the coordinator's ruling on build-lane serialisation, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/schedule.go"
remedy: "Let a dependent lane start once its predecessor is reviewed and landed on the run's own branch: open its pull request based on the predecessor's branch, merge the stack in order, and rebase the dependent lane (re-stamping its resolutions) when the predecessor changes"
related_intents: [itd-2610230856403803]
---

A build run's dependent lane cannot start until its predecessor has merged into the default branch: readiness reads a predecessor as landed only when its lane is done, its pull request an ancestor of the default branch, so a step that needs another waits a whole preflight and merge-queue cycle (about one to two and a half hours each on 2026-10-22) before it opens. Stacked changes are standard practice for dependent work (Graphite, Sapling and ghstack stacks, and merge queues taking a stack in order), and on 2026-10-22/23 that shape was rolled by hand twice: merge-gate steps 2 and 3, and model-matrix step 2 stacked on the route branch. Building it is more than a fix: the dependent lane's pull request is based on the predecessor's branch, the stack merges in order, and a lane rebases (and re-stamps what it resolved) when the predecessor changes.

2026-11-07 and 2026-11-08: seen again in the autonomous run of 7 Nov. At 19:41 on 7 Nov a two-step intent cost two preflights of two to five hours' wait each, and its second lane was cut from main rather than stacked on the first, though both edit the same file, so it had to merge main after the first landed. At 01:55 on 8 Nov a step could not be stacked on a lane held back until after a tag, so the spec's later steps waited for days; the only verb that closes a lane stuck off its reviewed head is --discard.
