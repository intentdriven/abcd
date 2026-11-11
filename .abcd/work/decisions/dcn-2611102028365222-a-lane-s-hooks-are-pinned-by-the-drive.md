---
id: dcn-2611102028365222
slug: a-lane-s-hooks-are-pinned-by-the-drive
date: 2026-11-10
---
2026-11-10 — A lane's hooks are pinned by the drive, never chosen by the lane: in a checkout with an origin the worktree stage asks origin itself for its default branch's tip, cuts the lane from that tip (refusing a default branch here that is neither the tip nor an ancestor of it, a tip not yet fetched, and a remote that cannot be asked), proves the lane's own hooks directory on disk is that commit's tree exactly before any commit in the lane, and records what it proved as the lane's part of the run's fingerprint, whose shape is unchanged. A lane opens on the tip rather than on an older ancestor, because an ancestor reached through a merge can hold hooks the default branch never held; a lane found again must hold the tip's hooks tree. With no origin or in a trial nothing is pinned and the lane is compared with the checkout as before. Left out and named in commands/implement.md: an absolute or global hooks path, the checkout's own files changing on a pull mid-run, and the watched settings each new run records afresh.
