---
schema_version: 1
id: "iss-2610261339023831"
slug: "re-running-a-failed-attribution-run"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/attribution.yml"
remedy: "Have the attribution step read the pull request's current title and body from the API (gh pr view, through the run's token) instead of the event payload, so any run of it, re-runs included, judges the text as it stands. Failing that, say in the step's failure message that a body fix is checked by the edited run, and that re-running the failed one replays the old body."
resolution: "The attribution workflow reads the pull request's current title and body from the API into files, and both form steps judge those files, so every run, re-runs included, judges the form as it stands; the concurrency group carries the head commit, so a re-run for an earlier head cannot cancel the current head's run."
impact: internal
---

Re-running a failed attribution run after fixing the pull-request body (gh run rerun on the old run) replays the event the run was started with, so it checks the old body again and fails. The re-run joins the attribution-<pr> concurrency group and cancels the newer run that the body edit started, which would have passed. On 2026-10-26, PRs #1129 and #1133 each sat red for about four hours: the edited runs 40325771175 and 40325732363 were cancelled, and runs 40298736534 and 40298846131 failed again on the bodies they started with.
