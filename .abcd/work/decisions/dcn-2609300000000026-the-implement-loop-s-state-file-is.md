---
id: dcn-2609300000000026
slug: the-implement-loop-s-state-file-is
date: 2026-09-30
---
2026-09-30 — The implement loop's state file is schema version 9 where a run works in parallel (ruling DR6, spc-2609202134341288), not the version 8 that spec's amendment names: the runner's record (itd-2609201916056194) reached the default branch first, on the v0.12.0 cut, and took version 8, so the two are numbered in the order they landed (integration integ24d of autonomous run A, `internal/core/implement/loop/state.go`). A version 7 or 8 file migrates its lane's one `awaiting` into `awaits` and is written back at version 9; a version 7 or 8 file carrying what only version 9 writes, and a version-9 file carrying `awaiting`, are refused naming the version and the key. The closed spec keeps its text; brief 34-build.md names the three versions.
