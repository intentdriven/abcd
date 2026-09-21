---
schema_version: 1
id: "iss-2609211105021406"
slug: "what-abcd-implement-would-have-to-carry-to-launch-the-pilot-run-without-its-prompt"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "itd-2609201916151817 (abcd implement); the pilot's run file and orchestrator prompt"
---

What abcd implement would have to carry to launch the pilot run without its prompt, from having run it: (1) the run file's part 1 as data: the design decisions, one per lane, that the run is not allowed to reopen, and a stop that names the decision it lacks; (2) the lane table: record ids, the files each touches, the sibling lanes touching the same files, the base sha, and the worktree path under the machine-scoped store; (3) the roles and models per role, with a fresh agent per fix round and a reviewer that never implemented; (4) the pace: window length, the pause as idle wake-ups, the agent ceiling with reviewers counted, and the tally fields written per session (lanes opened, finished, PRs, ceiling-wait minutes, agent-minutes, tokens); (5) the gate ladder per lane: preflight and fmt-check under the HOME alias where the username collides, the issue-resolution gate on the PR form, the push, the PR body re-read and lint outbound, auto-merge with the queue's method, and the rule that an armed PR is never pushed to; (6) the pre-authorised remedies, today the window recalibration, as named recipes the lane may run once; (7) the stop conditions as a list the orchestrator checks after every report; (8) the handover shape, NEXT.md's pilot section, so a crashed session is resumed from it (this run's orchestrator died once and was resumed from NEXT.md alone, the scratchpad and the run file both lost). Evidence from the sibling pilot on the same day, Dessau pilot 2 (its account is at that repository's for_abcd/pilot-2026-09-21-2-experiment.md): the outer loop ran as a script end to end in 82 minutes with zero stops for the work; every step that needed a hand was record-writing (Departures at close, captures for audit concerns, reviewers missing from the session ledger), and the pause never fired because the run fit one window, so itd-2609201925079472's pacing is untested past the first window. Implementer context peaked at 170k there against 349k in the first pilot.

## Evidence

- Dessau pilot 2 (2026-09-21, one intent through a scripted outer loop in 82 minutes): zero stops for the work; three script fixes; every step that needed a hand was record-writing; the pause never fired because the run fit one window. Implementer context peaked at 170k against 349k in the first pilot.
- Dessau pilot 3 (2026-09-21, 29 of 29 stop paths on fakes, then one intent through the loop with a planted merge-conflict fault): merged without a release because the lane was built before its parent, which the fix session captured as a major gating the cut. Part (g) of that pilot's account lists what the script did that the verb should own: run state plus a handover and a resume that re-enters on the same reason; the brief split and the JSON report schemas, held from the first session; Departures rendered at spec close from the lane reports; captures at audit ingest, checking the ledger for an existing record first; the auto-merge arm check after a push. The pacing gate never fired in any of the three pilots.
- This run (2026-09-20/21): the orchestrator session died once mid-gate and was resumed from the handover alone; a fresh worktree had no local-tier logs directory and a gate's redirect failed silently before make started; the forge's update-branch disarmed auto-merge; a merge-group check failed on a 504 fetching a tool; every one was a re-arm or a retry, none a new pull request.
