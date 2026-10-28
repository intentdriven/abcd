---
schema_version: 1
id: "iss-2610280911221843"
slug: "abcd-reserve-release-run-from-a-worktree"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Identify the holder by account, machine and root-commit key rather than worktree path, so a release from a sibling worktree of the same checkout succeeds and names the worktree that took the reservation."
---

abcd reserve --release run from a worktree other than the one that made the reservation is refused as held by your copy on another computer, though the holder is the same account on the same machine. Once that worktree is removed its reservations can only lapse. Two sessions met it, 25-28 Oct.
