---
schema_version: 1
id: "iss-2610290507526253"
slug: "the-own-copy-overlap-warning-of-abcd"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "keep the warning (itd-2610161016394967 rules a person's own reserve over their other copy goes ahead) and name the other copy in it: the sibling worktree on this machine by name, or a copy on another computer, and say when the worktrees could not be listed; the JSON own_overlap carries it as copy"
resolution: "the own-copy overlap warning names the other copy: the worktree on this machine by its path, a copy on another computer, or your other copy with why when the worktrees could not be listed (copy, copy_unread)"
impact: additive
---

The own-copy overlap warning of abcd reserve says only that the person holds the other record from their other copy, never which copy: a lane reserved over files a copy on another computer held with no sign the other work was elsewhere, as both lanes and the laptop's files read alike.
