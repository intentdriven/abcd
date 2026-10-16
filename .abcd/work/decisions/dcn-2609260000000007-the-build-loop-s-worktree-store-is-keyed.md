---
id: dcn-2609260000000007
slug: the-build-loop-s-worktree-store-is-keyed
date: 2026-09-26
---
2026-09-26 — The build loop's worktree store is keyed on the FULL root sha: a lane lives at `~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>` with the 40-hex root commit, the form the history, transcript and voyage stores use and the one the store's draft (itd-2609091014076309) specifies. The lanes of autonomous run A made by hand under the abbreviated key (`~/.abcd/worktrees/488a0aa9/<name>/`) are the pre-verb convention, not a second form of the store: the loop never reads or adopts a lane under that key, and those worktrees are retired with `git worktree remove` like any other (implementer of fix round fix2-loop2, autonomous run A, on item 4 of the loop2 review; spc-2609202134338445 piece 6).
