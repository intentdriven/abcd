---
schema_version: 1
id: "iss-2611010614204166"
slug: "under-facilitator-mode-auto-the"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "landing three build lanes overnight"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have the land stage record the review receipt for its own landing commit when that commit's only change is the spec's landed line for the step the validators passed, naming the validated head it builds on, so the merge gate arms the PR; or make the landed-line commit before validation so the validators judge the head that is pushed."
---

Under facilitator.mode auto the implement loop's land stage never arms a build lane's pull request: after the validators pass a head, the land stage commits its own record change (chore(record): land lane-N, marking the spec step landed) and pushes that commit as the PR head, and the merge gate then finds no review receipt naming the head, so nothing is armed. On 2026-11-01 PRs #1313, #1314 and #1315 each needed a second review pair over that one-line delta, recorded by hand with land --record-review, before auto-merge armed; an unattended run would stop there.
