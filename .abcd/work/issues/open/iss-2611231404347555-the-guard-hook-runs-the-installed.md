---
schema_version: 1
id: "iss-2611231404347555"
slug: "the-guard-hook-runs-the-installed"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/approve.go"
remedy: "when the hook cannot read a run's state because a newer abcd wrote it, judge a worktree removal on the evidence it can read (merged head, clean tree, no live process) and say which run it could not read, rather than refusing every removal; test with a run state one schema ahead"
---

The guard hook runs the installed release, which reads run state only up to an older schema; in a checkout whose runs were written by a source build it reports build runs unread and refuses every git worktree remove, even of a merged, clean worktree, so cleanup waits on the next release and an update.
