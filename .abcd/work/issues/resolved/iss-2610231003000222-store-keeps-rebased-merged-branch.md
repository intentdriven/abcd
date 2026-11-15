---
schema_version: 1
id: "iss-2610231003000222"
slug: "store-keeps-rebased-merged-branch"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/merged.go"
remedy: "When the tip is an ancestor of the default branch and the reflog shows no commit of the branch's own, ask the forge for a merged pull request whose head is that tip, and otherwise list the worktree as a quiet candidate (spec step 5) naming why, so a person can reclaim it; never reclaim on ancestry alone."
related_intents: [itd-2609091014076309]
resolution: "Delivered by spc-2609301811532881 as the remedy asks: the merged judgement asks the forge first for a merged pull request whose head is the local tip (internal/core/ahoy/worktree/merged.go:157; TestMergedJudgement 'forge head is the tip' and 'forge head is not the tip'), never reclaims on ancestry alone without a commit of the branch's own (merged.go:170; TestMergedJudgement 'never committed to'), and an unmerged worktree, the ancestor-tip one with no own commit included, becomes a quiet candidate once idle past worktrees.quiet_days with no open pull request (prune.go:253, quiet.go:142), listed with a dossier showing its commits not on the default branch (none here) and reclaimed only when named with --yes (TestPruneYesRemovesTheNamedQuietCandidateOnly, whose never-committed 'stay' worktree is the quiet row)."
impact: internal
resolved_by:
  spec: "spc-2609301811532881"
---

The worktree store's merged judgement keeps for good a branch whose merge it cannot see from the reflog or the forge: a branch rebased locally and then merged leaves no commit entry of its own in its reflog, and so does a merged branch whose reflog has expired; with no forge answer the content tests are always false once the tip is an ancestor of the default branch, so prune keeps the worktree on every run although its work has landed. Found by the Opus review of the store's steps 3 and 4 on 2026-10-23; it errs on the safe side, never reclaiming what it cannot prove.
