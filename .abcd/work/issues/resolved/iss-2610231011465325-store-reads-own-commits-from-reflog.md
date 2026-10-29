---
schema_version: 1
id: "iss-2610231011465325"
slug: "store-reads-own-commits-from-reflog"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/merged.go"
remedy: "Record the branch's start commit in the lane when the store adds the worktree (spec step 2), and judge the branch's own commits from that start instead of the reflog; a worktree the store did not add falls to the content tests."
related_intents: [itd-2609091014076309]
resolution: "The store's add records the branch's start commit in the lane, and the merged judgement reads a store-added branch's own commits from that start rather than its reflog, so a copy of a created branch, force-reset and left until its reflog expires, is not merged; TestMergedJudgementReadsAStoreBranchFromItsStart proves it, and TestMergedJudgementFallsBackWithoutAStartRecord holds the fallback for a worktree the store did not add."
impact: internal
---

The worktree store's merged judgement reads a branch's own commits from its reflog, and a reflog can carry another branch's history: a branch copied from a branch that was itself created, then force-reset and left until its unreachable reflog entries expire, keeps the source's creation entry and commit entries, so it is judged merged by ancestry with no commit of its own, and prune could remove its clean worktree with its ignored files. Every commit it holds is already on the default branch, so no commit is lost. Found by the Opus review of the store's steps 3 and 4 on 2026-10-23, after three earlier reflog shapes were closed.
