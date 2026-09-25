---
schema_version: 1
id: "iss-2609091829549814"
slug: "parallel-agents-share-one-scratch-path-and-mutate-each-other"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/principles"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Should AGENTS.md's scratch-copy convention name a private per-agent directory?"
---

Two agents working in separate worktrees were given the same session scratch directory and both used the same conventional subdirectory name for a mutation-testing copy, so one agent's mutation script rewrote the other's copy of the tree and restored it afterwards, which the second agent had no way to notice while it ran. Nothing was lost, because the mutation was reverted within the same run and neither worktree was touched, but the window is the hazard rather than the outcome: while a mutation is applied a gate reports on code nobody wrote, and the repository's own conventions say exactly that about mutating a tree, requiring the work to happen on a scratch copy and the tree to be proven clean before anything is reported. The convention covers the copy and not the collision, because it was written for one agent at a time and says nothing about where a scratch copy goes when several are running. The correction is cheap and belongs with the convention rather than in each brief: a scratch copy is made in a private directory the agent creates for itself, never at a shared conventional path, so two agents cannot pick the same one. Worth stating alongside it that a shared scratch directory is also where two agents can read each other's intermediate output and draw conclusions from work that is not theirs. Detector: a convention names the private-directory requirement where it names the scratch-copy requirement, and an agent brief that hands out a shared scratch path is a review finding.
