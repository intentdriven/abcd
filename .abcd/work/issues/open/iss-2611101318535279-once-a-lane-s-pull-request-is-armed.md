---
schema_version: 1
id: "iss-2611101318535279"
slug: "once-a-lane-s-pull-request-is-armed"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have step say the lane is waiting on its pull request's checks, naming the ones still pending, and exit with the waiting code rather than repeating the instruction; a test steps an armed lane with pending checks and finds a waiting line."
---

Once a lane's pull request is armed, implement step keeps printing the same instruction to take the lane's land stage, with no change in state and no word that it is waiting on checks. Seen on 7 Nov at 21:35 on a build lane while its pull request's checks ran.
