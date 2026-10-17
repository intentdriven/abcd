---
id: dcn-2610170728074867
slug: a-drain-move-reads-its-parked-lanes-from
date: 2026-10-17
---
2026-10-17 — A drain move reads its parked lanes from one listing of the person's own latest pull requests (facilitator-on-delegation, drain-many-accounts step 8): gh pr list --state all --author @me --limit 100, one page of the forge's API, matched to each parked lane by number and head branch; gh's pr list names no merge-queue field (ghcontract pins gh 2.100.0's list), so the arming (autoMergeRequest) stands for the queue entry; a pull request the listing does not show, or a listing that fails, keeps its lane renewed as before, so only an answer the forge gave changes a lane; one merged by a squash or rebase, its head not on the default branch, is named and no longer renewed; a lane handed back as closed keeps its worktree and branch for implement step --discard, which then closes nothing (grounds: one request a move, as dcn-2610161040380948 sets; fail-safe reading, where absence is never evidence of closure; iss-2610161015381750)
