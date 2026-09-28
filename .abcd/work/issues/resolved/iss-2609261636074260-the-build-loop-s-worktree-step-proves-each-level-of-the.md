---
schema_version: 1
id: "iss-2609261636074260"
slug: "the-build-loop-s-worktree-step-proves-each-level-of-the"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-loop2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/lane.go"
resolution: "ensureStore makes the worktree store's levels one at a time and holds each to fsutil.CallersAlone before making the next; a level it makes is 0700, and nothing is made inside a level another account owns or can write."
impact: internal
resolved_by:
  commit: "ae12c1bdd4527b27928137182421a98c17f1350f"
---

The build loop's worktree step proves each level of the machine-scoped worktree store (~/.abcd, worktrees, <root-sha>) a real directory through fsutil.EnsureRealDirAll but never applies fsutil.CallersAlone, so a pre-existing store level owned by another uid, or writable by group or other (mode 0777), is accepted and the lane's worktree is added into it, where another account could replace the checkout the implementer works in. Each existing level should be held to the caller-alone test the home-scoped declarations and the harness data directory use (iss-2609260057111315), and a level the step creates made 0700.

## Grounds

- pursued: only a store the caller alone can write holds a lane's checkout; TestTheWorktreeStepRefusesAStoreLevelAnyoneElseCanWrite would fail if a group- or world-writable level were accepted or had anything made inside it
