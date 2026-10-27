---
schema_version: 1
id: "iss-2610270815076960"
slug: "ci-s-linux-check-job-runs-within-about"
severity: "major"
category: "process"
source: "agent-observation"
found_during: "landing fix/catch-ci-failures-before-push (PR #1147)"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Raise the Linux check job's timeout-minutes to give the race stage a measured margin (for one, 45 against today's 28), or shard the race stage over internal/ across two jobs, and have make preflight warn when the race stage's local share of the limit nears it."
---

CI's Linux check job runs within about two minutes of its 30-minute timeout, so a branch that adds tests can be cancelled with no test failing. On 2026-10-27 the two successful Linux check jobs before 07:00Z took 28 minutes each, their race stage over internal/ about 25 of them. Pull request #1147's job was cancelled at 30m16s while its race stage was still printing results, though every gate and the race run had passed in make preflight. A cancelled required check costs a rerun, and the merge-queue entry runs the same job under the same limit.
