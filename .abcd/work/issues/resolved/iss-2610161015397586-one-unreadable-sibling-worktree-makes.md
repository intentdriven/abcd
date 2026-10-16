---
schema_version: 1
id: "iss-2610161015397586"
slug: "one-unreadable-sibling-worktree-makes"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/check.go"
remedy: "Refuse the move once, naming the unreadable worktree, instead of counting it as contention on every issue"
related_intents: [itd-2610161016394967]
resolution: "the drain refuses its move once, before any reservation or judgement, naming each sibling worktree or branch it cannot read and its repair, so an unreadable worktree no longer reads as contention on every issue"
impact: fix
resolved_by:
  intent: "itd-2610161016394967"
  spec: "spc-2610161044461448"
---

One unreadable sibling worktree makes the peers check fail for every issue, so a drain judges each issue, passes it over as contended, and ends with nothing eligible left; the primary checkout carries 108 worktrees, 46 locked by another tool.
