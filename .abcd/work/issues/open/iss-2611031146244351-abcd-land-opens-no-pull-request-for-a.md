---
schema_version: 1
id: "iss-2611031146244351"
slug: "abcd-land-opens-no-pull-request-for-a"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/land.go"
remedy: "Have land open the pull request from the body it composed when none is open (title from the branch's first commit, body re-read for a session URL or tool footer as the loop's land stage does), record any review receipts it is given, and arm in the same call."
---

abcd land opens no pull request for a branch made outside the build loop: it composes the body, pushes, and stops with 'no open pull request ... open it with the command, then run abcd land again'. On 2026-11-03 the coordinator opened eight pull requests that way by hand (gh pr create from the composed body, a re-read for footers, then land --record-review twice and land again to arm), in a helper script, so a CLI-only run cannot land a hand fix without a forge command of its own.
