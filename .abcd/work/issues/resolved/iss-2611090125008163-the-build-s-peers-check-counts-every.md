---
schema_version: 1
id: "iss-2611090125008163"
slug: "the-build-s-peers-check-counts-every"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run build lane"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/check.go"
remedy: "Count a local branch as holding a record in the start's peers check only when a worktree git lists has it checked out; name the others in the passing row instead of refusing on them."
resolution: "The start's peers check counts a local branch as holding a record only when a worktree has it checked out; a branch checked out nowhere is named in the passing row, and the drain's unreadable-peer refusal skips such a branch too."
impact: fix
---

The build's peers check counts every local branch as a peer holding a record, even a branch no worktree has checked out. Two hand-made backup branches that hold an intent in shipped/ make abcd build refuse that intent as held by a peer, though no session works on it.
