---
schema_version: 1
id: "iss-2610312044509977"
slug: "abcd-build-has-no-way-to-stop-a-run"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/schedule.go"
remedy: "Add a build option, read once at the start like --pace, naming the last spec step the run takes; after that step's lane lands, implement step takes its done stage, opens no further lane, releases the reservation and reports the run stopped, resumable by abcd build <id>."
---

abcd build has no way to stop a run after the lane in hand: once lane-1 of run-2610310228179894 landed as PR #1301, the next implement step opened lane-2 for the spec's next step before taking lane-1's done stage, so a session given one step had to stop stepping by hand and release the reservation with build --release, leaving the run parked with an empty lane-2 worktree.
