---
schema_version: 1
id: "iss-2611161232540905"
slug: "a-reservation-records-no-branch-only-the"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, review of the sweep's step 5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sweep/remote.go"
remedy: "Record the lane's branch in the reservation when it is made or renewed (a field in the reservation message of internal/core/claim, read by an older build as absent), and have the sweep hold a branch on origin that any live reservation records, from any machine or account; a test makes a reservation from another copy key naming a merged branch and finds the branch kept."
---

A reservation records no branch, only the copy that made it, so abcd sweep reads the branch a reservation holds from a local worktree of that copy: a reservation made on another machine or under another account holds no branch on origin, and a merged branch it still works on is kept only by the other proofs (no open pull request, merged at the judged tip, the lease). The holder's work is not lost, since the branch is merged, but its next push has to recreate the branch. The spec of itd-2610222214476887 asks for the reservation's own recorded branch, which does not exist yet (dcn-2611161144499939).
