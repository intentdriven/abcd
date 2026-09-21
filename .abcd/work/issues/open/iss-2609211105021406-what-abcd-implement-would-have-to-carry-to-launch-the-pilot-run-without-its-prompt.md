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
