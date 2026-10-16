---
schema_version: 1
id: "iss-2610161015387318"
slug: "the-host-judgement-is-asked-before-the"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "One fetch per move, skip held issues and held footprints before judging, and start each account at a different point within a category (a hash of the holder) (grounds: competing consumers with skip-locked selection; randomised work stealing)"
related_intents: [itd-2610161016394967]
resolution: "a drain reserves an issue before it asks the judgement over it, passes over an issue or footprint another copy holds in the move's view with no push, no judgement and no fetch, and takes each category-severity group file-scoped first in its own rendezvous-hash order"
impact: fix
resolved_by:
  intent: "itd-2610161016394967"
  spec: "spc-2610161044461448"
---

The host judgement is asked before the issue's reservation is checked, and an issue another account holds is retried on every move with a full check (plan, sibling worktree scan, fetch), so several drains all judge and retry the same head of the queue, since every drain takes the same order (category, severity, oldest).
