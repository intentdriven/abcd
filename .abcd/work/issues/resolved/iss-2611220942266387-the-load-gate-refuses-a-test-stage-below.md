---
schema_version: 1
id: "iss-2611220942266387"
slug: "the-load-gate-refuses-a-test-stage-below"
severity: "major"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_disk.go"
remedy: "Have the waiter run the paced cache trim (worktree.TrimGoCache) once before it refuses at the disk floor, and refuse only if the volume is still under the floor afterwards; make the trim a per-toolchain step chosen from the repository's language (Go's GOCACHE now, with the cache directory and keep rule declared for others), so the gate clears what abcd's own runs filled instead of waiting for a person."
resolution: "Partial by design: the waiter now runs the paced trim once before it refuses at the disk floor and refuses only if the volume is still low, saying on stderr what the trim freed or why it did not run, and the trim is a per-toolchain step told by the checkout's version files, with Go's build cache the one step; a declared cache directory and keep rule for other toolchains was left out and is captured as iss-2611221134035558."
impact: additive
resolved_by:
  commit: "f155e7ea8"
---

The load gate refuses a test stage below the disk floor, but the build cache trim that would clear it runs only when the build loop makes a lane or a worktree prune runs. A session that drives its own worktrees and preflights (an autonomous fix run on 21-22 Nov 2026) never reaches either, so the free disk fell from 30 GiB to 12 GiB over a night of preflights and was cleared by hand with find -delete, five times. The trim is also Go's alone; a repository in another language has no trim at all.
