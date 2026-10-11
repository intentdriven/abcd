---
schema_version: 1
id: "iss-2610050532499080"
slug: "spent-worktrees-and-branches-survive-the"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "/abcd:peers after the v0.13.0 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "~/.abcd.noindex/worktrees/"
remedy: "Add a post-cut cleanup step to the release runbook: remove every clean worktree and branch whose work is on origin/main, delete the remote branches of closed or merged pull requests, and put each unlanded branch to the person to keep or drop; itd-118 automates it later."
---

Nothing removed the spent worktrees and branches after the v0.13.0 cut (tagged 2026-10-05), so the checkout still carries work that is already on main, and abcd peers reports 35 live peers and 51 differing records, which buries what is actually in flight.

Spent, with the work on origin/main and the worktree clean:
- worktrees: bump (chore/bump-x-text), dr6Spec (docs/build-loop-parallel-lanes), drainGuard (fix/guard-default-word-set), run-B (detached), agentsSwitch (fix/one-conventions-file-own-switch), board-readability (docs/board-readability, landed under another commit), board2 (feat/board-renderer), e8termr3 (fix/guard-link-bound), postcut (chore/post-cut-captures), scan50 (chore/capture-test-home-spellings);
- local branches with no worktree: chore/capture-dependabot-gate, feat/audit-owed-flag, feat/loop-parallel-lanes, fix/arm-only-under-review, integ/land-24d, plan/consistency-followups, plan/worktree-store-review, release/v0.12.0, release/v0.12.0-r2, fix/scanner-selector-closing-call (superseded by #779);
- remote branches: origin/fix/scanner-selector-closing-call (#777 closed, superseded) and origin/docs/capture-sibling-repo-owner-redaction (#749 closed; its record iss-2609291409053602 is resolved on main, the condition its closing comment set for deleting the branch).

Unlanded and unruled: worktrees A, B, C, D, E1, E2, F and integration (the 2 September security-sweep lanes, 10 to 124 commits each, none on main) and 18 local branches with no worktree (phase-9/* and readings/opening-run, 18 to 190 commits each, the cold-reading Iteration 2 work). Nobody has said whether they are kept or dropped. The September worktrees also sit under the abbreviated store key 488a0aa9, which AGENTS.md says names a directory no verb reads.

itd-118 (draft) is the automated form of this cleanup; until it ships, the release cut has no step that does it by hand. Related: a record captured on 2 September and held only on those unlanded branches says nothing maintains the worktree convention.
