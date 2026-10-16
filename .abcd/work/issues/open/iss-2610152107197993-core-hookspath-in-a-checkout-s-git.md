---
schema_version: 1
id: "iss-2610152107197993"
slug: "core-hookspath-in-a-checkout-s-git"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "first real reservation push, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks"
remedy: "Point core.hooksPath at a relative .githooks so each worktree runs its own tree's hooks, or have the pre-push hook check that it is the hook of the tree being pushed and say so when it is not"
---

core.hooksPath in a checkout's .git/config is the primary checkout's absolute .githooks path, so every linked worktree runs the primary checkout's hooks rather than its own. When the primary checkout falls behind main (409 commits on 2026-10-15), a worktree on a current branch runs a stale pre-push hook: the first real reservation push was refused because the stale hook did not know the reservation namespace. Nothing reports the skew.
